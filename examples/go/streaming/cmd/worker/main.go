package main

import (
	"log"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/examples/go/streaming/workflows"
)

func main() {
	hatchet, err := v1.NewHatchetClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	// Create the streaming workflow
	streamingWorkflow := workflows.StreamingWorkflow(hatchet)

	// Create worker
	w, err := hatchet.Worker(worker.WorkerOpts{
		Name: "streaming-worker",
		Workflows: []workflow.WorkflowBase{
			streamingWorkflow,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	// Use interrupt context to handle Ctrl+C
	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	log.Println("Starting streaming worker...")
	err = w.StartBlocking(interruptCtx)
	if err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}