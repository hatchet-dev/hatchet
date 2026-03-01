package main

import (
	"context"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// > Step 03 Push Approval Event
// Include the runID so the event matches the specific task waiting for it.
func pushApproval(client *hatchet.Client, runID string, approved bool, reason string) {
	_ = client.Events().Push(context.Background(), "approval:decision", map[string]interface{}{
		"runId":    runID,
		"approved": approved,
		"reason":   reason,
	})
}
