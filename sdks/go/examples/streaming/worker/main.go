package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/examples/go/streaming/shared"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	streamingWorkflow := shared.StreamingWorkflow(client)

	w, err := client.NewWorker("streaming-worker", hatchet.WithWorkflows(streamingWorkflow))
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
