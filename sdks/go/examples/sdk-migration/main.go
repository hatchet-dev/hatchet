package main

import (
	"context"
	"log"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// Define input/output types
	type Input struct {
		Message string `json:"message"`
	}

	type Output struct {
		Result string `json:"result"`
	}

	// Create a simple task
	task := client.NewStandaloneTask("simple-task", func(ctx hatchet.Context, input Input) (Output, error) {
		return Output{Result: "Processed: " + input.Message}, nil
	})

	// Start worker
	worker, err := client.NewWorker("worker", hatchet.WithWorkflows(task))
	if err != nil {
		log.Fatal(err)
	}

	if err := worker.StartBlocking(context.Background()); err != nil {
		log.Fatal(err)
	}
}
