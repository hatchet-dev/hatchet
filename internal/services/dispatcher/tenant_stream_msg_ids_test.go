//go:build !e2e && !load && !rampup && !integration

package dispatcher

import (
	"maps"
	"slices"
	"testing"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

// TestTenantStreamMsgIDsInSync guards against drift between msgqueue's
// tenant-stream publish allowlist (tenantStreamMsgIDs, which gates what
// PubTenantMessage publishes to tenant topics) and the message IDs the
// dispatcher's gRPC streams actually consume (the keys of
// workflowEventConverters and workflowRunMatchers).
//
// The failure mode this prevents is silent: before the pub/sub split, every
// durable send was implicitly mirrored to the tenant stream, and making the
// mirror explicit nearly dropped task-cancelled on the API cancellation path
// (caught in review of #4480). A consumed ID missing from the allowlist means
// subscribers never see those events; an allowlisted ID with no consumer means
// every publish is wasted.
func TestTenantStreamMsgIDsInSync(t *testing.T) {
	consumed := make(map[string]struct{}, len(workflowEventConverters)+len(workflowRunMatchers))

	for id := range workflowEventConverters {
		consumed[id] = struct{}{}
	}

	for id := range workflowRunMatchers {
		consumed[id] = struct{}{}
	}

	published := msgqueue.TenantStreamMsgIDs()

	for _, id := range slices.Sorted(maps.Keys(consumed)) {
		if !slices.Contains(published, id) {
			t.Errorf(
				"the dispatcher's streams consume %q, but msgqueue's tenantStreamMsgIDs allowlist does not include it, so PubTenantMessage never publishes it and subscribers silently miss those events. Add %q to tenantStreamMsgIDs in internal/msgqueue/pubsub.go and make sure every producer of it routes through PubTenantMessage.",
				id, id,
			)
		}
	}

	for _, id := range published {
		if _, ok := consumed[id]; !ok {
			t.Errorf(
				"msgqueue's tenantStreamMsgIDs allowlist includes %q, but the dispatcher's streams do not consume it, so every publish of it is wasted. Either remove %q from tenantStreamMsgIDs in internal/msgqueue/pubsub.go, or add a handler for it to workflowEventConverters or workflowRunMatchers in internal/services/dispatcher/server.go.",
				id, id,
			)
		}
	}
}
