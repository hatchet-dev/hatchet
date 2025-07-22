package main

import (
	"context"
	"log"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	workflow := client.NewWorkflow("my-workflow")
	workflow.NewTask("my-task", func(ctx hatchet.Context, input any) (any, error) {
		return "Hello, World!", nil
	})

	worker, err := client.NewWorker("my-worker", hatchet.WithWorkflows(workflow))
	if err != nil {
		log.Fatalf("failed to create hatchet worker: %v", err)
	}

	if err := worker.StartBlocking(context.Background()); err != nil {
		log.Fatalf("failed to start hatchet worker: %v", err)
	}
}
