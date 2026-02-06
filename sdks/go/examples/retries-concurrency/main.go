package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type TaskInput struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Payload  string `json:"payload"`
}

type TaskOutput struct {
	ID          string `json:"id"`
	ProcessedAt string `json:"processed_at"`
	Result      string `json:"result"`
	Attempt     int    `json:"attempt"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	workflow := client.NewWorkflow("retry-concurrency-workflow")

	// Task with retries and concurrency control
	var maxRuns int32 = 2
	strategy := types.GroupRoundRobin

	_ = workflow.NewTask("unreliable-task", func(ctx hatchet.Context, input TaskInput) (TaskOutput, error) {
		attempt := ctx.RetryCount()
		log.Printf("Processing task %s (attempt %d)", input.ID, attempt)

		// Simulate unreliable processing - fails 70% of the time on first attempt
		if attempt == 0 && rand.Float32() < 0.7 { //nolint:gosec // This is a demo
			return TaskOutput{}, errors.New("simulated failure")
		}

		// Simulate some processing time
		time.Sleep(time.Duration(100+rand.Intn(400)) * time.Millisecond) //nolint:gosec // This is a demo

		return TaskOutput{
			ID:          input.ID,
			Attempt:     attempt,
			ProcessedAt: time.Now().Format(time.RFC3339),
			Result:      "Successfully processed " + input.Payload,
		}, nil
	},
		hatchet.WithRetries(3),
		hatchet.WithRetryBackoff(2.0, 60), // Exponential backoff: 2s, 4s, 8s, then cap at 60s
		hatchet.WithExecutionTimeout(30*time.Second),
		hatchet.WithConcurrency(&types.Concurrency{
			Expression:    "input.category", // Limit concurrency per category
			MaxRuns:       &maxRuns,         // Max 2 concurrent tasks per category
			LimitStrategy: &strategy,        // Round-robin distribution
		}),
	)

	worker, err := client.NewWorker("retry-worker",
		hatchet.WithWorkflows(workflow),
		hatchet.WithSlots(10), // Allow up to 10 concurrent tasks total
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Submit multiple tasks with different categories
	go func() {
		time.Sleep(2 * time.Second)

		categories := []string{"high-priority", "normal", "batch"}

		for i := 0; i < 10; i++ {
			category := categories[rand.Intn(len(categories))] //nolint:gosec // This is a demo

			_, err = client.RunNoWait(context.Background(), "retry-concurrency-workflow", TaskInput{
				ID:       fmt.Sprintf("task-%d", i),
				Category: category,
				Payload:  fmt.Sprintf("data for task %d", i),
			})
			if err != nil {
				log.Printf("failed to submit task %d: %v", i, err)
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	log.Println("Starting worker with retry and concurrency controls...")
	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
