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

	task := client.NewStandaloneTask("process-message", func(ctx hatchet.Context, input SimpleInput) (SimpleOutput, error) {
		return SimpleOutput{
			Result: "Processed: " + input.Message,
		}, nil
	})

	worker, err := client.NewWorker("simple-worker", hatchet.WithWorkflows(task))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	err = worker.StartBlocking(interruptCtx)
	if err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
