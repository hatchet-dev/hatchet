package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// > Declaring the tasks
type ParentInput struct {
	Count int `json:"count"`
}

type ParentOutput struct {
	Sum int `json:"sum"`
}

func Parent(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("parent-task",
		func(ctx hatchet.Context, input ParentInput) (ParentOutput, error) {
			log.Printf("Parent workflow spawning %d child workflows", input.Count)

			// Spawn multiple child workflows and collect results
			sum := 0
			for i := 0; i < input.Count; i++ {
				log.Printf("Spawning child workflow %d/%d", i+1, input.Count)

				// Spawn child workflow and wait for result
				childResult, err := Child(client).Run(ctx, ChildInput{
					Value: i + 1,
				})
				if err != nil {
					return ParentOutput{}, fmt.Errorf("failed to spawn child workflow %d: %w", i, err)
				}

				var childOutput ChildOutput
				err = childResult.Into(&childOutput)
				if err != nil {
					return ParentOutput{}, fmt.Errorf("failed to get child workflow result: %w", err)
				}

				sum += childOutput.Result

				log.Printf("Child workflow %d completed with result: %d", i+1, childOutput.Result)
			}

			log.Printf("All child workflows completed. Total sum: %d", sum)
			return ParentOutput{
				Sum: sum,
			}, nil
		},
	)
}

type ChildInput struct {
	Value int `json:"value"`
}

type ChildOutput struct {
	Result int `json:"result"`
}

func Child(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("child-task",
		func(ctx hatchet.Context, input ChildInput) (ChildOutput, error) {
			return ChildOutput{
				Result: input.Value * 2,
			}, nil
		},
	)
}


func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	parentWorkflow := Parent(client)
	childWorkflow := Child(client)

	// Create a worker with both workflows
	worker, err := client.NewWorker("child-workflow-worker",
		hatchet.WithWorkflows(childWorkflow, parentWorkflow),
		hatchet.WithSlots(10), // Allow parallel execution of child workflows
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Run the parent workflow
	go func() {
		// Wait a bit for worker to start
		for i := 0; i < 3; i++ {
			log.Printf("Starting in %d seconds...", 3-i)
			select {
			case <-context.Background().Done():
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}

		log.Println("Triggering parent workflow...")
		_, err := parentWorkflow.Run(context.Background(), ParentInput{
			Count: 5, // Spawn 5 child workflows
		})
		if err != nil {
			log.Printf("failed to run parent workflow: %v", err)
		}
	}()

	_ = func() error {
		var hCtx hatchet.Context
		// > Spawning a child workflow
		// Inside a parent task
		childResult, err := childWorkflow.Run(hCtx, ChildInput{
			Value: 1,
		})
		if err != nil {
			return err
		}


		_ = childResult

		n := 5

		// > Parallel child task execution
		// Run multiple child tasks in parallel using goroutines
		var wg sync.WaitGroup
		var mu sync.Mutex
		results := make([]*ChildOutput, 0, n)

		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(index int) {
				defer wg.Done()
				result, err := childWorkflow.Run(hCtx, ChildInput{Value: index})
				if err != nil {
					return
				}

				var childOutput ChildOutput
				err = result.Into(&childOutput)
				if err != nil {
					return
				}

				mu.Lock()
				results = append(results, &childOutput)
				mu.Unlock()
			}(i)
		}
		wg.Wait()

		// > Error handling
		result, err := childWorkflow.Run(hCtx, ChildInput{Value: 1})
		if err != nil {
			// Handle error from child workflow
			fmt.Printf("Child workflow failed: %v\n", err)
			// Decide how to proceed - retry, skip, or fail the parent
		}

		_ = result

		return nil
	}

	log.Println("Starting worker for child workflows demo...")
	log.Println("Features demonstrated:")
	log.Println("  - Parent workflow spawning multiple child workflows")
	log.Println("  - Child workflow execution and result collection")
	log.Println("  - Parallel child workflow processing")
	log.Println("  - Parent-child workflow communication")

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
