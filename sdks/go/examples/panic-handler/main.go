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

	task := client.NewStandaloneTask("panic-handler-task", func(ctx hatchet.Context, input any) (any, error) {
		panic("intentional panic for demonstration")
	})

	worker, err := client.NewWorker("panic-handler-worker", hatchet.WithPanicHandler(func(ctx hatchet.Context, recovered any) {
		log.Printf("panic handler: %v", recovered)
	}), hatchet.WithWorkflows(task))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	if err := worker.StartBlocking(context.Background()); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
