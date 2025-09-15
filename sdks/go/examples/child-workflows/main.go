package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type ParentInput struct {
	Count int `json:"count"`
}

type ChildInput struct {
	Value int `json:"value"`
}

type ChildOutput struct {
	Result int `json:"result"`
}

type ParentOutput struct {
	Sum int `json:"sum"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create child workflow
	childWorkflow := client.NewWorkflow("child-workflow",
		hatchet.WithWorkflowDescription("Child workflow that processes a single value"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	processValueTask := childWorkflow.NewTask("process-value", func(ctx hatchet.Context, input ChildInput) (ChildOutput, error) {
		log.Printf("Child workflow processing value: %d", input.Value)

		// Simulate some processing
		result := input.Value * 2

		return ChildOutput{
			Result: result,
		}, nil
	})

	// Create parent workflow that spawns multiple child workflows
	parentWorkflow := client.NewWorkflow("parent-workflow",
		hatchet.WithWorkflowDescription("Parent workflow that spawns child workflows"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	_ = parentWorkflow.NewTask("spawn-children", func(ctx hatchet.Context, input ParentInput) (ParentOutput, error) {
		log.Printf("Parent workflow spawning %d child workflows", input.Count)

		// Spawn multiple child workflows and collect results
		sum := 0
		for i := 0; i < input.Count; i++ {
			log.Printf("Spawning child workflow %d/%d", i+1, input.Count)

			// Spawn child workflow and wait for result
			childResult, err := childWorkflow.RunAsChild(ctx, ChildInput{
				Value: i + 1,
			}, hatchet.RunAsChildOpts{})
			if err != nil {
				return ParentOutput{}, fmt.Errorf("failed to spawn child workflow %d: %w", i, err)
			}

			var childOutput ChildOutput
			taskResult := childResult.TaskOutput(processValueTask.GetName())
			err = taskResult.Into(&childOutput)
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
	})

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
		_, err := client.Run(context.Background(), "parent-workflow", ParentInput{
			Count: 5, // Spawn 5 child workflows
		})
		if err != nil {
			log.Printf("failed to run parent workflow: %v", err)
		}
	}()

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
