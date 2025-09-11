package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/examples/go/streaming/shared"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	v1worker "github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
)

func main() {
	hatchet, err := v1.NewHatchetClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	streamingWorkflow := shared.StreamingWorkflow(hatchet)

	w, err := hatchet.Worker(v1worker.WorkerOpts{
		Name: "streaming-worker",
		Workflows: []workflow.WorkflowBase{
			streamingWorkflow,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	log.Println("Starting streaming worker...")

	if err := w.StartBlocking(interruptCtx); err != nil {
		log.Println("Worker failed:", err)
	}
}
