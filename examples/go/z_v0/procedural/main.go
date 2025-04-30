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

const NUM_CHILDREN = 50

type proceduralChildInput struct {
	Index int `json:"index"`
}

type proceduralChildOutput struct {
	Index int `json:"index"`
}

type proceduralParentOutput struct {
	ChildSum int `json:"child_sum"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	events := make(chan string, 5*NUM_CHILDREN)
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
			Description: "This is a test of procedural workflows.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(
					func(ctx worker.HatchetContext) (result *proceduralParentOutput, err error) {
						childWorkflows := make([]*client.Workflow, NUM_CHILDREN)

						for i := 0; i < NUM_CHILDREN; i++ {
							childInput := proceduralChildInput{
								Index: i,
							}

							childWorkflow, err := ctx.SpawnWorkflow("procedural-child-workflow", childInput, &worker.SpawnWorkflowOpts{
								AdditionalMetadata: &map[string]string{
									"childKey": "childValue",
								},
							})

							if err != nil {
								return nil, err
							}

							childWorkflows[i] = childWorkflow

							events <- fmt.Sprintf("child-%d-started", i)
						}

						eg := errgroup.Group{}

						eg.SetLimit(NUM_CHILDREN)

						childOutputs := make([]int, 0)
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
									childOutputs = append(childOutputs, childOutput.Index)
									childOutputsMu.Unlock()

									events <- fmt.Sprintf("child-%d-completed", childOutput.Index)

									return nil

								}
							}(i, childWorkflow))
						}

						finishedCh := make(chan struct{})

						go func() {
							defer close(finishedCh)
							err = eg.Wait()
						}()

						timer := time.NewTimer(60 * time.Second)

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
									if childOutput == i {
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

						for _, childOutput := range childOutputs {
							sum += childOutput
						}

						return &proceduralParentOutput{
							ChildSum: sum,
						}, nil
					},
				).SetTimeout("10m"),
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
			Description: "This is a test of procedural workflows.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(
					func(ctx worker.HatchetContext) (result *proceduralChildOutput, err error) {
						input := proceduralChildInput{}

						err = ctx.WorkflowInput(&input)

						if err != nil {
							return nil, err
						}

						return &proceduralChildOutput{
							Index: input.Index,
						}, nil
					},
				).SetName("step-one"),
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}

	go func() {
		time.Sleep(1 * time.Second)

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
