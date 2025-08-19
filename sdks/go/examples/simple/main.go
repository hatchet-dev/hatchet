package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type SimpleInput struct {
	Message string `json:"message"`
}

type SimpleOutput struct {
	Result string `json:"result"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create a simple workflow with one task
	workflow := client.NewWorkflow("simple-workflow")
	_ = workflow.NewTask("process-message", func(ctx hatchet.Context, input SimpleInput) (SimpleOutput, error) {
		return SimpleOutput{
			Result: "Processed: " + input.Message,
		}, nil
	})

	// Create a worker to run the workflow
	worker, err := client.NewWorker("simple-worker", hatchet.WithWorkflows(workflow))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	// Start the worker (blocks)
	err = worker.StartBlocking(interruptCtx)
	if err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
