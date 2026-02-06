package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type ProcessInput struct {
	Message string `json:"message"`
	ID      int    `json:"id"`
}

type ProcessOutput struct {
	Result string `json:"result"`
	ID     int    `json:"id"`
}

func main() {
	// Create a new Hatchet client
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create a workflow for bulk processing
	workflow := client.NewWorkflow("bulk-processing-workflow")

	// Define the processing task
	workflow.NewTask("process-item", func(ctx hatchet.Context, input ProcessInput) (ProcessOutput, error) {
		// Simulate some processing work
		time.Sleep(time.Duration(100+input.ID*50) * time.Millisecond)

		log.Printf("Processing item %d: %s", input.ID, input.Message)

		return ProcessOutput{
			ID:     input.ID,
			Result: fmt.Sprintf("Processed item %d: %s", input.ID, input.Message),
		}, nil
	})

	// Create a worker to run the workflow
	worker, err := client.NewWorker("bulk-operations-worker", hatchet.WithWorkflows(workflow))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	// Start the worker in a goroutine
	go func() {
		log.Println("Starting bulk operations worker...")
		if err := worker.StartBlocking(interruptCtx); err != nil {
			log.Printf("worker failed: %v", err)
		}
	}()

	// Wait a moment for the worker to start
	time.Sleep(2 * time.Second)

	// Prepare bulk data
	bulkInputs := make([]ProcessInput, 10)
	for i := 0; i < 10; i++ {
		bulkInputs[i] = ProcessInput{
			ID:      i + 1,
			Message: fmt.Sprintf("Task number %d", i+1),
		}
	}

	log.Printf("Running bulk operations with %d items...", len(bulkInputs))

	// > Bulk run tasks
	// Prepare inputs as []RunManyOpt for bulk run
	inputs := make([]hatchet.RunManyOpt, len(bulkInputs))
	for i, input := range bulkInputs {
		inputs[i] = hatchet.RunManyOpt{
			Input: input,
		}
	}

	// Run workflows in bulk
	ctx := context.Background()
	runRefs, err := workflow.RunMany(ctx, inputs)
	if err != nil {
		log.Fatalf("failed to run bulk workflows: %v", err)
	}

	runIDs := make([]string, len(runRefs))
	for i, runRef := range runRefs {
		runIDs[i] = runRef.RunId
	}

	log.Printf("Started %d bulk workflows with run IDs: %v", len(runRefs), runRefs)

	// Optionally monitor some of the runs
	for i, runID := range runIDs {
		if i < 3 { // Monitor first 3 runs as examples
			log.Printf("Monitoring run %d with ID: %s", i+1, runID)
		}
	}

	log.Println("All bulk operations started. Press Ctrl+C to stop the worker.")

	<-interruptCtx.Done()
}
