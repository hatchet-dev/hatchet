package main

import (
	"context"
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type EventInput struct {
	Message string `json:"message"`
	Source  string `json:"source"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Define Event Task
	task := client.NewStandaloneTask("process-event", func(ctx hatchet.Context, input EventInput) (map[string]string, error) {
		source := input.Source
		if source == "" {
			source = "api"
		}
		return map[string]string{"processed": input.Message, "source": source}, nil
	}, hatchet.WithWorkflowEvents("order:created", "user:signup"))
	// !!

	// > Step 02 Register Event Trigger
	// Push an event from your app. Call this from your webhook handler or API.
	pushEvent := func() {
		_ = client.Events().Push(context.Background(), "order:created", map[string]interface{}{
			"message": "Order #1234",
			"source":  "webhook",
		})
	}
	_ = pushEvent
	// !!

	// > Step 04 Run Worker
	worker, err := client.NewWorker("event-driven-worker", hatchet.WithWorkflows(task))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
	// !!
}
