package main

import (
	"context"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

type PriorityInput struct {
	UserID    string `json:"user_id"`
	TaskType  string `json:"task_type"`
	Message   string `json:"message"`
	IsPremium bool   `json:"is_premium"`
}

type PriorityOutput struct {
	ProcessedAt      time.Time `json:"processed_at"`
	ProcessedMessage string    `json:"processed_message"`
	UserID           string    `json:"user_id"`
	Priority         int32     `json:"priority"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	_ = func() error {
		// > Default priority
		workflow := client.NewWorkflow(
			"priority",
			hatchet.WithWorkflowDefaultPriority(features.RunPriorityLow),
		)

		// > Running a task with priority
		ref, err := client.RunNoWait(
			context.Background(),
			workflow.GetName(),
			PriorityInput{},
			hatchet.WithRunPriority(features.RunPriorityLow),
		)
		if err != nil {
			return err
		}

		_ = ref

		// > Schedule and cron
		priority := features.RunPriorityHigh

		schedule, err := client.Schedules().Create(
			context.Background(),
			workflow.GetName(),
			features.CreateScheduledRunTrigger{
				Priority: &priority,
			},
		)
		if err != nil {
			return err
		}

		cron, err := client.Crons().Create(
			context.Background(),
			workflow.GetName(),
			features.CreateCronTrigger{
				Priority: &priority,
			},
		)
		if err != nil {
			return err
		}

		_ = schedule
		_ = cron

		return nil
	}

	// Create workflow that demonstrates priority-based processing
	priorityWorkflow := client.NewWorkflow("priority-demo",
		hatchet.WithWorkflowDescription("Demonstrates priority-based task processing"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// Task that processes requests based on priority
	_ = priorityWorkflow.NewTask("process-request", func(ctx hatchet.Context, input PriorityInput) (PriorityOutput, error) {
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
	worker, err := client.NewWorker("priority-worker",
		hatchet.WithWorkflows(priorityWorkflow),
		hatchet.WithSlots(5), // Allow parallel processing
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Function to run workflow with specific priority
	runWithPriority := func(priority hatchet.RunPriority, input PriorityInput, delay time.Duration) {
		time.Sleep(delay)
		log.Printf("Submitting %s task with priority %d for user %s",
			input.TaskType, priority, input.UserID)

		// Run workflow with specific priority
		_, err := client.Run(context.Background(), "priority-demo", input, hatchet.WithRunPriority(priority))
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
		go runWithPriority(features.RunPriorityLow, PriorityInput{
			UserID:    "user-001",
			TaskType:  "report",
			Message:   "Generate monthly report",
			IsPremium: false,
		}, 0)

		// Submit high priority task second (but should be processed first)
		go runWithPriority(features.RunPriorityHigh, PriorityInput{
			UserID:    "user-002",
			TaskType:  "alert",
			Message:   "Critical system alert",
			IsPremium: true,
		}, 100*time.Millisecond)

		// Submit medium priority task third
		go runWithPriority(features.RunPriorityMedium, PriorityInput{
			UserID:    "user-003",
			TaskType:  "notification",
			Message:   "User notification",
			IsPremium: false,
		}, 200*time.Millisecond)

		// Submit another high priority task
		go runWithPriority(features.RunPriorityHigh, PriorityInput{
			UserID:    "user-004",
			TaskType:  "emergency",
			Message:   "Emergency response needed",
			IsPremium: true,
		}, 300*time.Millisecond)

		// Submit more tasks to show queuing behavior
		go runWithPriority(features.RunPriorityLow, PriorityInput{
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

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
