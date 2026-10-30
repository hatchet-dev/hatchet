package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/examples/go/durable-streams/shared"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	w, err := client.NewWorker(
		"durable-streams-worker",
		hatchet.WithWorkflows(shared.ProgressTask(client), shared.RetryingProgressTask(client)),
	)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := w.StartBlocking(interruptCtx); err != nil {
		log.Println("Worker failed:", err)
	}
}
