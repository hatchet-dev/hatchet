package main

import (
	"context"
	"log"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type EventInput struct {
	UserID  string `json:"user_id"`
	Action  string `json:"action"`
	Payload any    `json:"payload"`
}

type ProcessOutput struct {
	ProcessedAt string `json:"processed_at"`
	UserID      string `json:"user_id"`
	Action      string `json:"action"`
	Result      string `json:"result"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create an event-triggered workflow
	workflow := client.NewWorkflow("event-workflow", 
		hatchet.WithWorkflowEvents("user:created", "user:updated"),
	)

	workflow.NewTask("process-user-event", func(ctx hatchet.Context, input EventInput) (ProcessOutput, error) {
		log.Printf("Processing %s event for user %s", input.Action, input.UserID)
		
		// Access the original event data
		eventData := ctx.FilterPayload()
		log.Printf("Event data: %+v", eventData)
		
		return ProcessOutput{
			ProcessedAt: time.Now().Format(time.RFC3339),
			UserID:      input.UserID,
			Action:      input.Action,
			Result:      "Event processed successfully",
		}, nil
	})

	worker, err := client.NewWorker("event-worker", hatchet.WithWorkflows(workflow))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Simulate sending events in a separate goroutine
	go func() {
		time.Sleep(2 * time.Second) // Wait for worker to start

		// Send a user:created event
		log.Println("Sending user:created event...")
		// Note: In a real application, you would use the client's event API
		// This is just for demonstration
		
		// Simulate another event after a delay
		time.Sleep(5 * time.Second)
		log.Println("Sending user:updated event...")
	}()

	log.Println("Starting event worker...")
	if err := worker.Run(context.Background()); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}