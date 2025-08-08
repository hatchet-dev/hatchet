package main

import (
	"context"
	"log"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type PriorityInput struct {
	UserID    string `json:"user_id"`
	TaskType  string `json:"task_type"`
	Message   string `json:"message"`
	IsPremium bool   `json:"is_premium"`
}

type PriorityOutput struct {
	ProcessedMessage string    `json:"processed_message"`
	Priority         int32     `json:"priority"`
	ProcessedAt      time.Time `json:"processed_at"`
	UserID           string    `json:"user_id"`
}

func main() {
	clientInstance, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create workflow that demonstrates priority-based processing
	priorityWorkflow := clientInstance.NewWorkflow("priority-demo",
		hatchet.WithWorkflowDescription("Demonstrates priority-based task processing"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// Task that processes requests based on priority
	priorityWorkflow.NewTask("process-request", func(ctx hatchet.Context, input PriorityInput) (PriorityOutput, error) {
		// Access the current priority from context
		currentPriority := ctx.Priority()
		
		log.Printf("Processing %s request for user %s with priority %d", 
			input.TaskType, input.UserID, currentPriority)

		// Simulate different processing times based on priority
		// Higher priority = faster processing
		var processingTime time.Duration
		switch {
		case currentPriority >= 3: // High priority
			processingTime = 500 * time.Millisecond
			log.Printf("High priority processing for user %s", input.UserID)
		case currentPriority >= 2: // Medium priority
			processingTime = 2 * time.Second
			log.Printf("Medium priority processing for user %s", input.UserID)
		default: // Low priority
			processingTime = 5 * time.Second
			log.Printf("Low priority processing for user %s", input.UserID)
		}

		// Simulate processing work
		time.Sleep(processingTime)

		processedMessage := ""
		if input.IsPremium {
			processedMessage = "PREMIUM: " + input.Message
		} else {
			processedMessage = "STANDARD: " + input.Message
		}

		log.Printf("Completed processing for user %s with priority %d", input.UserID, currentPriority)

		return PriorityOutput{
			ProcessedMessage: processedMessage,
			Priority:         currentPriority,
			ProcessedAt:      time.Now(),
			UserID:           input.UserID,
		}, nil
	})

	// Create a worker to process the workflows
	worker, err := clientInstance.NewWorker("priority-worker",
		hatchet.WithWorkflows(priorityWorkflow),
		hatchet.WithSlots(5), // Allow parallel processing
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Function to run workflow with specific priority
	runWithPriority := func(priority int32, input PriorityInput, delay time.Duration) {
		time.Sleep(delay)
		log.Printf("Submitting %s task with priority %d for user %s", 
			input.TaskType, priority, input.UserID)
		
		// Run workflow with specific priority
		err := clientInstance.RunWithPriority(context.Background(), "priority-demo", input, priority)
		if err != nil {
			log.Printf("Failed to run workflow with priority %d: %v", priority, err)
		}
	}

	// Run multiple workflows with different priorities to demonstrate priority handling
	go func() {
		time.Sleep(2 * time.Second)

		log.Println("\n=== Priority Demo Started ===")
		log.Println("Submitting tasks in order: Low, High, Medium priority")
		log.Println("Watch the processing order - high priority should be processed first!")

		// Submit low priority task first
		go runWithPriority(1, PriorityInput{
			UserID:    "user-001",
			TaskType:  "report",
			Message:   "Generate monthly report",
			IsPremium: false,
		}, 0)

		// Submit high priority task second (but should be processed first)
		go runWithPriority(4, PriorityInput{
			UserID:    "user-002",
			TaskType:  "alert",
			Message:   "Critical system alert",
			IsPremium: true,
		}, 100*time.Millisecond)

		// Submit medium priority task third
		go runWithPriority(2, PriorityInput{
			UserID:    "user-003",
			TaskType:  "notification",
			Message:   "User notification",
			IsPremium: false,
		}, 200*time.Millisecond)

		// Submit another high priority task
		go runWithPriority(5, PriorityInput{
			UserID:    "user-004",
			TaskType:  "emergency",
			Message:   "Emergency response needed",
			IsPremium: true,
		}, 300*time.Millisecond)

		// Submit more tasks to show queuing behavior
		go runWithPriority(1, PriorityInput{
			UserID:    "user-005",
			TaskType:  "backup",
			Message:   "System backup",
			IsPremium: false,
		}, 400*time.Millisecond)
	}()

	log.Println("Starting worker for priority demo...")
	log.Println("Features demonstrated:")
	log.Println("  - Task priority configuration")
	log.Println("  - Priority-based processing order")
	log.Println("  - Accessing current priority in task context")
	log.Println("  - Different processing behavior based on priority")
	log.Println("  - Premium vs standard user handling")

	if err := worker.Run(context.Background()); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}