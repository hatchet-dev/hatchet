package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// Global depth configuration
type DepthConfig struct {
	MaxDepth  int         `json:"max_depth"`
	Branching map[int]int `json:"branching"` // depth -> number of children
}

// Default configuration: 3 levels with decreasing branching factor
var GlobalDepthConfig = DepthConfig{
	MaxDepth: 3,
	Branching: map[int]int{
		0: 5, // Root level spawns 5 children
		1: 3, // Level 1 spawns 3 children each
		2: 2, // Level 2 spawns 2 children each
	},
}

type proceduralChildInput struct {
	Index      int         `json:"index"`
	Depth      int         `json:"depth"`
	Config     DepthConfig `json:"config"`
	ParentPath string      `json:"parent_path"`
}

type proceduralChildOutput struct {
	Index      int `json:"index"`
	Depth      int `json:"depth"`
	ChildSum   int `json:"child_sum"`
	TotalNodes int `json:"total_nodes"`
}

type proceduralParentOutput struct {
	ChildSum   int `json:"child_sum"`
	TotalNodes int `json:"total_nodes"`
}

// Helper function to calculate maximum possible nodes
func calculateMaxNodes(config DepthConfig) int {
	total := 1 // Root node
	currentLevelNodes := 1

	for depth := 0; depth < config.MaxDepth; depth++ {
		if branching, exists := config.Branching[depth]; exists && branching > 0 {
			currentLevelNodes *= branching
			total += currentLevelNodes
		} else {
			break
		}
	}
	return total
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	// Calculate max possible events based on config
	maxEvents := calculateMaxNodes(GlobalDepthConfig) * 2 // *2 for start/complete events
	events := make(chan string, maxEvents)
	interrupt := cmdutils.InterruptChan()

	cleanup, err := run(events)
	if err != nil {
		panic(err)
	}

	<-interrupt

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}
}

func run(events chan<- string) (func() error, error) {
	c, err := client.New()

	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating worker: %w", err)
	}

	testSvc := w.NewService("test")

	err = testSvc.On(
		worker.NoTrigger(),
		&worker.WorkflowJob{
			Name:        "procedural-parent-workflow",
			Description: "This is a test of procedural workflows with hierarchical depth.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(
					func(ctx worker.HatchetContext) (result *proceduralParentOutput, err error) {
						// Root level starts at depth 0
						numChildren := GlobalDepthConfig.Branching[0]
						if numChildren == 0 {
							return &proceduralParentOutput{ChildSum: 0, TotalNodes: 1}, nil
						}

						childWorkflows := make([]*client.Workflow, numChildren)

						for i := 0; i < numChildren; i++ {
							childInput := proceduralChildInput{
								Index:      i,
								Depth:      1, // Children start at depth 1
								Config:     GlobalDepthConfig,
								ParentPath: "root",
							}

							childWorkflow, err := ctx.SpawnWorkflow("procedural-child-workflow", childInput, &worker.SpawnWorkflowOpts{
								AdditionalMetadata: &map[string]string{
									"childKey":   "childValue",
									"depth":      fmt.Sprintf("%d", 1),
									"parentPath": "root",
								},
							})

							if err != nil {
								return nil, err
							}

							childWorkflows[i] = childWorkflow

							events <- fmt.Sprintf("root-child-%d-started", i)
						}

						eg := errgroup.Group{}
						eg.SetLimit(numChildren)

						childOutputs := make([]proceduralChildOutput, 0)
						childOutputsMu := sync.Mutex{}

						for i, childWorkflow := range childWorkflows {
							eg.Go(func(i int, childWorkflow *client.Workflow) func() error {
								return func() error {
									childResult, err := childWorkflow.Result()

									if err != nil {
										return err
									}

									childOutput := proceduralChildOutput{}

									err = childResult.StepOutput("step-one", &childOutput)

									if err != nil {
										return err
									}

									childOutputsMu.Lock()
									childOutputs = append(childOutputs, childOutput)
									childOutputsMu.Unlock()

									events <- fmt.Sprintf("root-child-%d-completed", childOutput.Index)

									return nil
								}
							}(i, childWorkflow))
						}

						finishedCh := make(chan struct{})

						go func() {
							defer close(finishedCh)
							err = eg.Wait()
						}()

						timer := time.NewTimer(120 * time.Second) // Increased timeout for deeper hierarchy

						select {
						case <-finishedCh:
							if err != nil {
								return nil, err
							}
						case <-timer.C:
							incomplete := make([]int, 0)
							// print non-complete children
							for i := range childWorkflows {
								completed := false
								for _, childOutput := range childOutputs {
									if childOutput.Index == i {
										completed = true
										break
									}
								}

								if !completed {
									incomplete = append(incomplete, i)
								}
							}

							return nil, fmt.Errorf("timed out waiting for the following child workflows to complete: %v", incomplete)
						}

						sum := 0
						totalNodes := 1 // Count root node

						for _, childOutput := range childOutputs {
							sum += childOutput.ChildSum
							totalNodes += childOutput.TotalNodes
						}

						fmt.Printf("🎯 Parent workflow completed: ChildSum=%d, TotalNodes=%d\n", sum, totalNodes)

						return &proceduralParentOutput{
							ChildSum:   sum,
							TotalNodes: totalNodes,
						}, nil
					},
				).SetTimeout("15m"),
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}

	err = testSvc.On(
		worker.NoTrigger(),
		&worker.WorkflowJob{
			Name:        "procedural-child-workflow",
			Description: "This is a hierarchical child workflow that can spawn its own children.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(
					func(ctx worker.HatchetContext) (result *proceduralChildOutput, err error) {
						input := proceduralChildInput{}

						err = ctx.WorkflowInput(&input)

						if err != nil {
							return nil, err
						}

						childSum := input.Index // Start with own index
						totalNodes := 1         // Count self

						fmt.Printf("🌲 Node at depth %d (index %d, path %s.%d) starting\n",
							input.Depth, input.Index, input.ParentPath, input.Index)

						// Check if we should spawn children at this depth
						numChildren, shouldSpawn := input.Config.Branching[input.Depth]
						if shouldSpawn && input.Depth < input.Config.MaxDepth {
							fmt.Printf("🌱 Spawning %d children at depth %d\n", numChildren, input.Depth+1)

							// Spawn children recursively
							childWorkflows := make([]*client.Workflow, numChildren)

							for i := 0; i < numChildren; i++ {
								childInput := proceduralChildInput{
									Index:      i,
									Depth:      input.Depth + 1,
									Config:     input.Config,
									ParentPath: fmt.Sprintf("%s.%d", input.ParentPath, input.Index),
								}

								childWorkflow, err := ctx.SpawnWorkflow("procedural-child-workflow", childInput, &worker.SpawnWorkflowOpts{
									AdditionalMetadata: &map[string]string{
										"childKey":   "childValue",
										"depth":      fmt.Sprintf("%d", input.Depth+1),
										"parentPath": fmt.Sprintf("%s.%d", input.ParentPath, input.Index),
									},
								})

								if err != nil {
									return nil, err
								}

								childWorkflows[i] = childWorkflow

								events <- fmt.Sprintf("%s.%d-child-%d-started", input.ParentPath, input.Index, i)
							}

							// Wait for all children to complete
							eg := errgroup.Group{}
							eg.SetLimit(numChildren)

							childOutputs := make([]proceduralChildOutput, 0)
							childOutputsMu := sync.Mutex{}

							for i, childWorkflow := range childWorkflows {
								eg.Go(func(i int, childWorkflow *client.Workflow) func() error {
									return func() error {
										childResult, err := childWorkflow.Result()

										if err != nil {
											return err
										}

										childOutput := proceduralChildOutput{}

										err = childResult.StepOutput("step-one", &childOutput)

										if err != nil {
											return err
										}

										childOutputsMu.Lock()
										childOutputs = append(childOutputs, childOutput)
										childOutputsMu.Unlock()

										events <- fmt.Sprintf("%s.%d-child-%d-completed", input.ParentPath, input.Index, childOutput.Index)

										return nil
									}
								}(i, childWorkflow))
							}

							err = eg.Wait()
							if err != nil {
								return nil, err
							}

							// Aggregate child results
							for _, childOutput := range childOutputs {
								childSum += childOutput.ChildSum
								totalNodes += childOutput.TotalNodes
							}

							fmt.Printf("🌳 Node at depth %d completed with %d children: ChildSum=%d, TotalNodes=%d\n",
								input.Depth, len(childOutputs), childSum, totalNodes)
						} else {
							fmt.Printf("🍃 Leaf node at depth %d (max depth reached or no branching configured)\n", input.Depth)
						}

						return &proceduralChildOutput{
							Index:      input.Index,
							Depth:      input.Depth,
							ChildSum:   childSum,
							TotalNodes: totalNodes,
						}, nil
					},
				).SetName("step-one").SetTimeout("10m"),
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}

	go func() {
		time.Sleep(1 * time.Second)

		fmt.Printf("🚀 Starting hierarchical workflow with config: MaxDepth=%d, Branching=%v\n",
			GlobalDepthConfig.MaxDepth, GlobalDepthConfig.Branching)
		fmt.Printf("📊 Expected total nodes: %d\n", calculateMaxNodes(GlobalDepthConfig))

		_, err := c.Admin().RunWorkflow("procedural-parent-workflow", nil)

		if err != nil {
			panic(fmt.Errorf("error running workflow: %w", err))
		}
	}()

	cleanup, err := w.Start()

	if err != nil {
		panic(err)
	}

	return cleanup, nil
}
