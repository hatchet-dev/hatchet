package main

import (
	"context"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// > Step 03 Push Approval Event
// Your frontend or API pushes the approval event when the human clicks Approve/Reject.
func pushApproval(client *hatchet.Client) {
	_ = client.Events().Push(context.Background(), "approval:decision", map[string]interface{}{
		"approved": true,
		"reason":   "",
	})
}
