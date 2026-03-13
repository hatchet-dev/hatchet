package main

import (
	"context"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// > Step 03 Push Event
// Push an event to trigger the workflow. Use the same key as WithWorkflowEvents.
func pushEvent(client *hatchet.Client) {
	_ = client.Events().Push(context.Background(), "order:created", map[string]interface{}{
		"message": "Order #1234",
		"source":  "webhook",
	})
}

// !!
