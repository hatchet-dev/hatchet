package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type StickyInput struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
	Step      int    `json:"step"`
}

type StickyOutput struct {
	SessionID   string `json:"session_id"`
	WorkerID    string `json:"worker_id"`
	ProcessedAt string `json:"processed_at"`
	Message     string `json:"message"`
	Step        int    `json:"step"`
}

type SessionState struct {
	SessionID   string `json:"session_id"`
	WorkerID    string `json:"worker_id"`
	StepCount   int    `json:"step_count"`
	LastMessage string `json:"last_message"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create session workflow that maintains state on the same worker
	sessionWorkflow := client.NewWorkflow("session-demo",
		hatchet.WithWorkflowDescription("Demonstrates sticky session processing"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// Step 1: Initialize session
	step1 := sessionWorkflow.NewTask("initialize-session", func(ctx hatchet.Context, input StickyInput) (SessionState, error) {
		workerID := ctx.Worker().ID()
		log.Printf("[Worker %s] Initializing session %s", workerID, input.SessionID)

		return SessionState{
			SessionID:   input.SessionID,
			WorkerID:    workerID,
			StepCount:   1,
			LastMessage: input.Message,
		}, nil
	})

	// Step 2: Process session (runs on same worker)
	sessionWorkflow.NewTask("process-session", func(ctx hatchet.Context, input StickyInput) (SessionState, error) {
		workerID := ctx.Worker().ID()

		// Get previous step's output
		var sessionState SessionState
		if err := ctx.StepOutput("initialize-session", &sessionState); err != nil {
			return SessionState{}, fmt.Errorf("failed to get session state: %w", err)
		}

		log.Printf("[Worker %s] Processing session %s (was initialized on worker %s)",
			workerID, input.SessionID, sessionState.WorkerID)

		// Update session state
		sessionState.StepCount++
		sessionState.LastMessage = input.Message

		return sessionState, nil
	}, hatchet.WithParents(step1))

	// Child workflow for sticky child execution
	childWorkflow := client.NewWorkflow("sticky-child-demo",
		hatchet.WithWorkflowDescription("Child workflow that runs on same worker as parent"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	childWorkflow.NewTask("child-task", func(ctx hatchet.Context, input StickyInput) (StickyOutput, error) {
		workerID := ctx.Worker().ID()
		log.Printf("[Worker %s] Processing child task for session %s", workerID, input.SessionID)

		return StickyOutput{
			SessionID:   input.SessionID,
			WorkerID:    workerID,
			ProcessedAt: time.Now().Format(time.RFC3339),
			Message:     "CHILD: " + input.Message,
			Step:        input.Step,
		}, nil
	})

	// Parent workflow that spawns sticky child workflows
	parentWorkflow := client.NewWorkflow("sticky-parent-demo",
		hatchet.WithWorkflowDescription("Parent workflow that spawns sticky child workflows"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	parentWorkflow.NewTask("spawn-sticky-children", func(ctx hatchet.Context, input StickyInput) ([]StickyOutput, error) {
		workerID := ctx.Worker().ID()
		log.Printf("[Worker %s] Parent spawning sticky children for session %s", workerID, input.SessionID)

		var results []StickyOutput

		// Spawn multiple child workflows that should all run on the same worker
		for i := 1; i <= 3; i++ {
			log.Printf("[Worker %s] Spawning sticky child %d", workerID, i)

			sticky := true
			childResult, err := ctx.SpawnWorkflow("sticky-child-demo", StickyInput{
				SessionID: input.SessionID,
				Message:   fmt.Sprintf("Child %d message", i),
				Step:      i,
			}, &worker.SpawnWorkflowOpts{
				Key:    stringPtr(fmt.Sprintf("child-%s-%d", input.SessionID, i)),
				Sticky: &sticky, // This ensures child runs on same worker
			})
			if err != nil {
				return nil, fmt.Errorf("failed to spawn sticky child %d: %w", i, err)
			}

			// Wait for child to complete
			result, err := childResult.Result()
			if err != nil {
				return nil, fmt.Errorf("sticky child %d failed: %w", i, err)
			}

			var childOutput StickyOutput
			if err := result.StepOutput("child-task", &childOutput); err != nil {
				return nil, fmt.Errorf("failed to parse child %d output: %w", i, err)
			}

			log.Printf("[Worker %s] Child %d completed on worker %s", workerID, i, childOutput.WorkerID)
			results = append(results, childOutput)
		}

		return results, nil
	})

	// Comparison workflow that spawns non-sticky children
	nonStickyWorkflow := client.NewWorkflow("non-sticky-demo",
		hatchet.WithWorkflowDescription("Demonstrates non-sticky child workflow execution"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	nonStickyWorkflow.NewTask("spawn-regular-children", func(ctx hatchet.Context, input StickyInput) ([]StickyOutput, error) {
		workerID := ctx.Worker().ID()
		log.Printf("[Worker %s] Parent spawning non-sticky children for session %s", workerID, input.SessionID)

		var results []StickyOutput

		// Spawn child workflows without sticky flag (may run on different workers)
		for i := 1; i <= 3; i++ {
			log.Printf("[Worker %s] Spawning regular child %d", workerID, i)

			childResult, err := ctx.SpawnWorkflow("sticky-child-demo", StickyInput{
				SessionID: input.SessionID,
				Message:   fmt.Sprintf("Non-sticky child %d message", i),
				Step:      i,
			}, &worker.SpawnWorkflowOpts{
				Key: stringPtr(fmt.Sprintf("non-sticky-child-%s-%d", input.SessionID, i)),
				// No Sticky flag - children may run on different workers
			})
			if err != nil {
				return nil, fmt.Errorf("failed to spawn regular child %d: %w", i, err)
			}

			result, err := childResult.Result()
			if err != nil {
				return nil, fmt.Errorf("regular child %d failed: %w", i, err)
			}

			var childOutput StickyOutput
			if err := result.StepOutput("child-task", &childOutput); err != nil {
				return nil, fmt.Errorf("failed to parse regular child %d output: %w", i, err)
			}

			log.Printf("[Worker %s] Regular child %d completed on worker %s", workerID, i, childOutput.WorkerID)
			results = append(results, childOutput)
		}

		return results, nil
	})

	// Create multiple workers to demonstrate sticky behavior
	worker1, err := client.NewWorker("sticky-worker-1",
		hatchet.WithWorkflows(sessionWorkflow, childWorkflow, parentWorkflow, nonStickyWorkflow),
		hatchet.WithSlots(10),
	)
	if err != nil {
		log.Fatalf("failed to create worker 1: %v", err)
	}

	worker2, err := client.NewWorker("sticky-worker-2",
		hatchet.WithWorkflows(sessionWorkflow, childWorkflow, parentWorkflow, nonStickyWorkflow),
		hatchet.WithSlots(10),
	)
	if err != nil {
		log.Fatalf("failed to create worker 2: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	// Start both workers
	go func() {
		log.Println("Starting worker 1...")
		if err := worker1.StartBlocking(interruptCtx); err != nil {
			log.Printf("Worker 1 failed: %v", err)
		}
	}()

	go func() {
		log.Println("Starting worker 2...")
		if err := worker2.StartBlocking(interruptCtx); err != nil {
			log.Printf("Worker 2 failed: %v", err)
		}
	}()

	// Run demonstrations
	go func() {
		time.Sleep(3 * time.Second)

		log.Println("\n=== Session Workflow Demo ===")
		log.Println("Running multi-step workflow that should stay on same worker...")
		_, err := client.Run(context.Background(), "session-demo", StickyInput{
			SessionID: "session-001",
			Message:   "Initialize my session",
			Step:      1,
		})
		if err != nil {
			log.Printf("Session workflow error: %v", err)
		}

		time.Sleep(3 * time.Second)

		log.Println("\n=== Sticky Parent-Child Demo ===")
		log.Println("Parent spawning sticky children - all should run on same worker...")
		_, err = client.Run(context.Background(), "sticky-parent-demo", StickyInput{
			SessionID: "sticky-session-001",
			Message:   "Sticky parent message",
			Step:      1,
		})
		if err != nil {
			log.Printf("Sticky parent workflow error: %v", err)
		}

		time.Sleep(5 * time.Second)

		log.Println("\n=== Non-Sticky Comparison Demo ===")
		log.Println("Parent spawning regular children - may distribute across workers...")
		_, err = client.Run(context.Background(), "non-sticky-demo", StickyInput{
			SessionID: "regular-session-001",
			Message:   "Regular parent message",
			Step:      1,
		})
		if err != nil {
			log.Printf("Non-sticky workflow error: %v", err)
		}
	}()

	log.Println("Starting sticky workers demo...")
	log.Println("Features demonstrated:")
	log.Println("  - Multi-step workflows running on same worker")
	log.Println("  - Sticky child workflow execution")
	log.Println("  - Worker ID access in task context")
	log.Println("  - Session state maintenance across steps")
	log.Println("  - Comparison with non-sticky execution")

	<-interruptCtx.Done()

	time.Sleep(2 * time.Second)
}

func stringPtr(s string) *string {
	return &s
}
