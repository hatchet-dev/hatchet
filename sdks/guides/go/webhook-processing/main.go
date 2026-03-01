package main

import (
	"context"
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type WebhookPayload struct {
	EventID string                 `json:"event_id"`
	Type    string                 `json:"type"`
	Data    map[string]interface{} `json:"data"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Define Webhook Task
	task := client.NewStandaloneTask("process-webhook", func(ctx hatchet.Context, input WebhookPayload) (map[string]string, error) {
		return map[string]string{"processed": input.EventID, "type": input.Type}, nil
	}, hatchet.WithWorkflowEvents("webhook:stripe", "webhook:github"))
	// !!

	// > Step 02 Register Webhook
	// Call from your webhook endpoint to trigger the task.
	forwardWebhook := func(eventKey string, payload map[string]interface{}) {
		_ = client.Events().Push(context.Background(), eventKey, payload)
	}
	_ = forwardWebhook
	// !!

	// > Step 03 Process Payload
	// Validate event_id for deduplication; process idempotently.
	validateAndProcess := func(input WebhookPayload) (map[string]string, error) {
		if input.EventID == "" {
			return nil, nil // or return error
		}
		return map[string]string{"processed": input.EventID, "type": input.Type}, nil
	}
	_ = validateAndProcess
	// !!

	// > Step 04 Run Worker
	worker, err := client.NewWorker("webhook-worker", hatchet.WithWorkflows(task))
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
