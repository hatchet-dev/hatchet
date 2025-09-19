package main

import (
	"context"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type TimeoutInput struct {
	Message string `json:"message"`
}

type TimeoutOutput struct {
	Status    string `json:"status"`
	Completed bool   `json:"completed"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create workflow with timeout example
	timeoutWorkflow := client.NewWorkflow("timeout-demo",
		hatchet.WithWorkflowDescription("Demonstrates task timeout handling"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// > Execution Timeout
	// Task that will timeout - sleeps for 10 seconds but has 3 second timeout
	_ = timeoutWorkflow.NewTask("timeout-task",
		func(ctx hatchet.Context, input TimeoutInput) (TimeoutOutput, error) {
			log.Printf("Starting task that will timeout. Message: %s", input.Message)

			// Sleep for 10 seconds (will be interrupted by timeout)
			time.Sleep(10 * time.Second)

			// This should not be reached due to timeout
			log.Println("Task completed successfully (this shouldn't be reached)")
			return TimeoutOutput{
				Status:    "completed",
				Completed: true,
			}, nil
		},
		hatchet.WithExecutionTimeout(3*time.Second), // 3 second timeout
	)
	// !!

	// > Refresh Timeout
	// Create workflow with timeout refresh example
	refreshTimeoutWorkflow := client.NewWorkflow("refresh-timeout-demo",
		hatchet.WithWorkflowDescription("Demonstrates timeout refresh functionality"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// Task that refreshes its timeout to avoid timing out
	_ = refreshTimeoutWorkflow.NewTask("refresh-timeout-task",
		func(ctx hatchet.Context, input TimeoutInput) (TimeoutOutput, error) {
			log.Printf("Starting task with timeout refresh. Message: %s", input.Message)

			// Refresh timeout by 10 seconds
			log.Println("Refreshing timeout by 10 seconds...")
			err := ctx.RefreshTimeout("10s")
			if err != nil {
				log.Printf("Failed to refresh timeout: %v", err)
				return TimeoutOutput{
					Status:    "failed",
					Completed: false,
				}, err
			}

			// Now sleep for 5 seconds (should complete successfully)
			log.Println("Sleeping for 5 seconds...")
			time.Sleep(5 * time.Second)

			log.Println("Task completed successfully after timeout refresh")
			return TimeoutOutput{
				Status:    "completed",
				Completed: true,
			}, nil
		},
		hatchet.WithExecutionTimeout(3*time.Second), // Initial 3 second timeout
	)
	// !!

	// Create workflow with context cancellation handling
	cancellationWorkflow := client.NewWorkflow("cancellation-timeout-demo",
		hatchet.WithWorkflowDescription("Demonstrates proper context cancellation handling"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// Task that properly handles context cancellation
	_ = cancellationWorkflow.NewTask("cancellation-aware-task",
		func(ctx hatchet.Context, input TimeoutInput) (TimeoutOutput, error) {
			log.Printf("Starting cancellation-aware task. Message: %s", input.Message)

			// Loop with context cancellation checking
			for i := 0; i < 10; i++ {
				select {
				case <-ctx.Done():
					log.Printf("Task cancelled/timed out after %d iterations", i)
					return TimeoutOutput{
						Status:    "cancelled",
						Completed: false,
					}, nil
				default:
					log.Printf("Working... iteration %d/10", i+1)
					time.Sleep(1 * time.Second)
				}
			}

			log.Println("Task completed successfully")
			return TimeoutOutput{
				Status:    "completed",
				Completed: true,
			}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Second), // 5 second timeout
	)

	// Create a worker with all workflows
	worker, err := client.NewWorker("timeout-worker",
		hatchet.WithWorkflows(timeoutWorkflow, refreshTimeoutWorkflow, cancellationWorkflow),
		hatchet.WithSlots(3),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Run workflow instances to demonstrate timeouts
	go func() {
		time.Sleep(2 * time.Second)

		// Demo 1: Task that will timeout
		log.Println("\n=== Demo 1: Task Timeout ===")
		_, err := client.Run(context.Background(), "timeout-demo", TimeoutInput{
			Message: "This task will timeout after 3 seconds",
		})
		if err != nil {
			log.Printf("Expected timeout error: %v", err)
		}

		time.Sleep(2 * time.Second)

		// Demo 2: Task that refreshes timeout
		log.Println("\n=== Demo 2: Timeout Refresh ===")
		_, err = client.Run(context.Background(), "refresh-timeout-demo", TimeoutInput{
			Message: "This task will refresh its timeout and complete",
		})
		if err != nil {
			log.Printf("Refresh timeout error: %v", err)
		}

		time.Sleep(2 * time.Second)

		// Demo 3: Cancellation-aware task
		log.Println("\n=== Demo 3: Cancellation-Aware Task ===")
		_, err = client.Run(context.Background(), "cancellation-timeout-demo", TimeoutInput{
			Message: "This task will handle cancellation gracefully",
		})
		if err != nil {
			log.Printf("Cancellation-aware task error: %v", err)
		}
	}()

	log.Println("Starting worker for timeout demos...")
	log.Println("Features demonstrated:")
	log.Println("  - Task execution timeouts")
	log.Println("  - Timeout refresh functionality")
	log.Println("  - Context cancellation handling")
	log.Println("  - Graceful timeout handling")

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
