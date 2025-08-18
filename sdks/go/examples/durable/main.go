package main

import (
	"context"
	"log"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type DurableInput struct {
	Message string `json:"message"`
	Delay   int    `json:"delay"` // seconds
}

type DurableOutput struct {
	ProcessedAt string `json:"processed_at"`
	Message     string `json:"message"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create a workflow with a durable task that can sleep
	workflow := client.NewWorkflow("durable-workflow")

	durableTask := workflow.NewDurableTask("long-running-task", func(ctx hatchet.DurableContext, input DurableInput) (DurableOutput, error) {
		log.Printf("Starting task, will sleep for %d seconds", input.Delay)

		// Durable sleep - this can be resumed if the worker restarts
		if _, err := ctx.SleepFor(time.Duration(input.Delay) * time.Second); err != nil {
			return DurableOutput{}, err
		}

		log.Printf("Finished sleeping, processing message: %s", input.Message)

		return DurableOutput{
			ProcessedAt: time.Now().Format(time.RFC3339),
			Message:     "Processed: " + input.Message,
		}, nil
	})
	_ = durableTask // Durable task reference available

	worker, err := client.NewWorker("durable-worker",
		hatchet.WithWorkflows(workflow),
		hatchet.WithDurableSlots(10), // Allow up to 10 concurrent durable tasks
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Run the workflow with a 30-second delay
	_, err = client.Run(context.Background(), "durable-workflow", DurableInput{
		Message: "Hello from durable task!",
		Delay:   30,
	})
	if err != nil {
		log.Fatalf("failed to run workflow: %v", err)
	}

	log.Println("Workflow started. Worker will process it...")
	if err := worker.StartBlocking(); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
