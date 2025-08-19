package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type CancellationInput struct {
	Message string `json:"message"`
}

type CancellationOutput struct {
	Status    string `json:"status"`
	Completed bool   `json:"completed"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create a workflow that demonstrates cancellation handling
	workflow := client.NewWorkflow("cancellation-demo",
		hatchet.WithWorkflowDescription("Demonstrates workflow cancellation patterns"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// Add a long-running task that can be cancelled
	_ = workflow.NewTask("long-running-task", func(ctx hatchet.Context, input CancellationInput) (CancellationOutput, error) {
		log.Printf("Starting long-running task with message: %s", input.Message)

		// Simulate long-running work with cancellation checking
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				log.Printf("Task cancelled after %d steps", i)
				return CancellationOutput{
					Status:    "cancelled",
					Completed: false,
				}, nil
			default:
				log.Printf("Working... step %d/10", i+1)
				time.Sleep(1 * time.Second)
			}
		}

		log.Println("Task completed successfully")
		return CancellationOutput{
			Status:    "completed",
			Completed: true,
		}, nil
	}, hatchet.WithTimeout(30*time.Second))

	// Create a worker
	worker, err := client.NewWorker("cancellation-worker",
		hatchet.WithWorkflows(workflow),
		hatchet.WithSlots(3),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Run workflow instances to demonstrate cancellation
	go func() {
		time.Sleep(2 * time.Second)

		log.Println("Starting workflow instance...")
		ref, err := client.RunNoWait(context.Background(), "cancellation-demo", CancellationInput{
			Message: "This task will run for 10 seconds and can be cancelled",
		})
		if err != nil {
			log.Printf("failed to run workflow: %v", err)
		}

		// Send cancellation after 2 seconds
		time.Sleep(2 * time.Second)

		_, err = client.Runs().Cancel(context.Background(), rest.V1CancelTaskRequest{
			ExternalIds: &[]types.UUID{uuid.MustParse(ref.RunId)},
		})
		if err != nil {
			log.Printf("failed to cancel workflow: %v", err)
		}
	}()

	log.Println("Starting worker for cancellation demo...")
	log.Println("Features demonstrated:")
	log.Println("  - Long-running task with cancellation checking")
	log.Println("  - Context cancellation handling")
	log.Println("  - Graceful shutdown on cancellation")
	log.Println("  - Task timeout configuration")

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
