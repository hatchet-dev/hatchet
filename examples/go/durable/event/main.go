package main

import (
	"context"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type DurableInput struct {
	Message string `json:"message"`
	Delay   int    `json:"delay"` // seconds
}

type DurableOutput struct {
	ProcessedAt string `json:"processed_at"`
	Message     string `json:"message"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Durable Event
	task := client.NewStandaloneDurableTask("long-running-task", func(ctx hatchet.DurableContext, input DurableInput) (DurableOutput, error) {
		log.Printf("Starting task, will sleep for %d seconds", input.Delay)

		if _, err := ctx.WaitForEvent("user:updated", ""); err != nil {
			return DurableOutput{}, err
		}

		log.Printf("Finished waiting for event, processing message: %s", input.Message)

		return DurableOutput{
			ProcessedAt: time.Now().Format(time.RFC3339),
			Message:     "Processed: " + input.Message,
		}, nil
	})

	_ = func(ctx hatchet.DurableContext) (DurableOutput, error) {
		// > Durable Event With Filter
		if _, err := ctx.WaitForEvent("user:updated", "input.status_code == 200"); err != nil {
			return DurableOutput{}, err
		}

		return DurableOutput{}, nil
	}

	worker, err := client.NewWorker("durable-worker",
		hatchet.WithWorkflows(task),
		hatchet.WithDurableSlots(10),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	go func() {
		if err := worker.StartBlocking(interruptCtx); err != nil {
			log.Fatalf("failed to start worker: %v", err)
		}
	}()

	// Run the workflow with a 30-second delay
	_, err = client.Run(context.Background(), "durable-workflow", DurableInput{
		Message: "Hello from durable task!",
		Delay:   30,
	})
	if err != nil {
		log.Fatalf("failed to run workflow: %v", err)
	}

	<-interruptCtx.Done()
}
