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

	// Create an event-triggered standalone task
	workflow := client.NewWorkflow("process-user-event",
		hatchet.WithWorkflowEvents("user:created", "user:updated"),
	)

	workflow.NewTask("process-user-event", func(ctx hatchet.Context, input EventInput) (ProcessOutput, error) {
		log.Printf("Processing %s event for user %s", input.Action, input.UserID)
		log.Printf("Event payload contains: %+v", input.Payload)

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

	// Send events in a separate goroutine
	go func() {
		time.Sleep(3 * time.Second) // Wait for worker to start

		// Send a user:created event
		log.Println("Sending user:created event...")
		err := client.Events().Push(context.Background(), "user:created", EventInput{
			UserID: "user-123",
			Action: "created",
			Payload: map[string]any{
				"email": "user@example.com",
				"name":  "John Doe",
			},
		})
		if err != nil {
			log.Printf("Failed to send user:created event: %v", err)
		}

		// Send another event after a delay
		time.Sleep(5 * time.Second)
		log.Println("Sending user:updated event...")
		err = client.Events().Push(context.Background(), "user:updated", EventInput{
			UserID: "user-123",
			Action: "updated",
			Payload: map[string]any{
				"email": "newemail@example.com",
				"name":  "John Doe",
			},
		})
		if err != nil {
			log.Printf("Failed to send user:updated event: %v", err)
		}
	}()

	log.Println("Starting event worker...")
	log.Println("Features demonstrated:")
	log.Println("  - Event-triggered standalone tasks")
	log.Println("  - Processing event payloads")
	log.Println("  - Real event sending and handling")
	if err := worker.StartBlocking(); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
