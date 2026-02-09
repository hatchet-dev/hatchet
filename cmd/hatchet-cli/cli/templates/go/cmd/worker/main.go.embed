package main

import (
	"log"

	"github.com/hatchet-dev/hatchet-go-quickstart/client"
	"github.com/hatchet-dev/hatchet-go-quickstart/workflows"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	c, err := client.HatchetClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	worker, err := c.NewWorker(
		"first-workflow-worker",
		hatchet.WithWorkflows(workflows.FirstWorkflow(c)),
	)
	if err != nil {
		log.Fatalf("Failed to create Hatchet worker: %v", err)
	}

	// we construct an interrupt context to handle Ctrl+C
	// you can pass in your own context.Context here to the worker
	interruptCtx, cancel := cmdutils.NewInterruptContext()

	defer cancel()

	err = worker.StartBlocking(interruptCtx)
	if err != nil {
		log.Fatalf("Failed to start Hatchet worker: %v", err)
	}
}
