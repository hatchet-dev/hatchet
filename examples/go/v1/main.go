package main

import (
	"context"
	"fmt"
	"log"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type MessageInput struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

type ProcessedOutput struct {
	OriginalMessage string    `json:"original_message"`
	ProcessedAt     time.Time `json:"processed_at"`
	UserID          string    `json:"user_id"`
	Result          string    `json:"result"`
}

func main() {
	// Create a new Hatchet client
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create a workflow with type-safe input/output
	workflow := client.NewWorkflow("message-processor",
		hatchet.WithWorkflowDescription("Processes user messages with enhanced capabilities"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// Add a task with proper typing and configuration
	workflow.NewTask("process-message", 
		func(ctx hatchet.Context, input MessageInput) (ProcessedOutput, error) {
			log.Printf("Processing message from user %s: %s", input.UserID, input.Message)
			
			// Simulate some processing time
			time.Sleep(100 * time.Millisecond)
			
			return ProcessedOutput{
				OriginalMessage: input.Message,
				ProcessedAt:     time.Now(),
				UserID:          input.UserID,
				Result:          fmt.Sprintf("Processed: %s", input.Message),
			}, nil
		},
		hatchet.WithRetries(3),
		hatchet.WithTimeout(30*time.Second),
	)

	// Create a worker with the workflow
	worker, err := client.NewWorker("message-worker", 
		hatchet.WithWorkflows(workflow),
		hatchet.WithSlots(5), // Allow 5 concurrent tasks
	)
	if err != nil {
		log.Fatalf("failed to create hatchet worker: %v", err)
	}

	// Run a workflow instance to demonstrate
	go func() {
		time.Sleep(2 * time.Second) // Wait for worker to start
		
		log.Println("Triggering workflow execution...")
		err := client.Run(context.Background(), "message-processor", MessageInput{
			Message: "Hello from the new Hatchet Go SDK v1!",
			UserID:  "demo-user-123",
		})
		if err != nil {
			log.Printf("failed to run workflow: %v", err)
		}
	}()

	log.Println("Starting Hatchet worker...")
	log.Println("✨ New SDK Features Demonstrated:")
	log.Println("  - Type-safe input/output")
	log.Println("  - Clean, reflection-based API")
	log.Println("  - No generics required")
	log.Println("  - Functional options pattern")
	log.Println("  - Built-in retry and timeout configuration")

	if err := worker.Run(context.Background()); err != nil {
		log.Fatalf("failed to start hatchet worker: %v", err)
	}
}
