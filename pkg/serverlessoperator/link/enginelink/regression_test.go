//go:build !e2e && !load && !rampup && !integration

package enginelink

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

func entryCompleted(branch, node int64) *v1.DurableTaskResponse {
	return &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EntryCompleted{
		EntryCompleted: &v1.DurableTaskEventLogEntryCompletedResponse{Ref: &v1.DurableEventLogEntryRef{BranchId: branch, NodeId: node}},
	}}
}

// The dispatcher registers a durable task for routing before the register-worker handshake
// completes, so an entry_completed restored for a resumed invocation can reach the channel
// before the handshake ack. It must be held for the invocation, not rejected as an invalid
// handshake, and delivered after the ack that names it.
func TestEarlyCompletionDoesNotBreakHandshake(t *testing.T) {
	h := newHarness(t)
	h.dispatcher.durableEarlyResponses = []*v1.DurableTaskResponse{entryCompleted(3, 7)}

	reg, _ := h.open(t, link.OpenOpts{})

	ch, err := reg.OpenDurable(context.Background(), uuid.NewString(), 1)
	require.NoError(t, err, "an entry_completed arriving before the register_worker ack must not fail the handshake")

	t.Cleanup(func() { _ = ch.Close() })

	// The held entry is delivered once the ack naming its ref goes out.
	require.NoError(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_WaitFor{
		WaitFor: &v1.DurableTaskWaitForRequest{},
	}}))

	h.dispatcher.respond(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_WaitForAck{
		WaitForAck: &v1.DurableTaskEventWaitForAckResponse{Ref: &v1.DurableEventLogEntryRef{BranchId: 3, NodeId: 7}},
	}})

	first := recvWithin(t, ch)
	require.NotNil(t, first.GetWaitForAck(), "the ack comes first")

	second := recvWithin(t, ch)
	require.NotNil(t, second.GetEntryCompleted(), "then the entry held since before the handshake")
	assert.Equal(t, int64(7), second.GetEntryCompleted().GetRef().GetNodeId())
}

// The handshake buffer is bounded: an engine flooding the channel before the ack fails the
// open instead of growing without limit.
func TestHandshakeBufferIsBounded(t *testing.T) {
	h := newHarness(t)

	early := make([]*v1.DurableTaskResponse, 0, handshakeHoldLimit+1)

	for i := 0; i <= handshakeHoldLimit; i++ {
		early = append(early, entryCompleted(1, int64(i)))
	}

	h.dispatcher.durableEarlyResponses = early

	reg, _ := h.open(t, link.OpenOpts{})

	_, err := reg.OpenDurable(context.Background(), uuid.NewString(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "handshake")
}

// A Send blocked because the engine is not reading must return when the channel closes.
func TestBlockedSendUnblocksOnClose(t *testing.T) {
	h := newHarness(t)
	h.dispatcher.durableStallAfterHandshake = true

	reg, _ := h.open(t, link.OpenOpts{})

	ch, err := reg.OpenDurable(context.Background(), uuid.NewString(), 1)
	require.NoError(t, err)

	blocked := make(chan error, 1)

	go func() {
		blocked <- ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_CompleteMemo{
			CompleteMemo: &v1.DurableTaskCompleteMemoRequest{Ref: &v1.DurableEventLogEntryRef{NodeId: 1}},
		}})
	}()

	select {
	case err := <-blocked:
		t.Fatalf("send should block while the engine is not reading: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	require.NoError(t, ch.Close())

	select {
	case err := <-blocked:
		assert.ErrorIs(t, err, link.ErrChannelClosed)
	case <-time.After(2 * time.Second):
		t.Fatal("a blocked Send survived Close")
	}
}

// A large initial action set is linked in chunks of maxActionsPerDelta through the bulk
// path, and the worker is activated only once every chunk is in.
func TestOpenLinksInitialActionsInChunks(t *testing.T) {
	h := newHarness(t)

	actions := make([]string, 0, 2500)

	for i := 0; i < 2500; i++ {
		actions = append(actions, fmt.Sprintf("ns_svc:action%04d", i))
	}

	h.workers.activateHook = func() {
		h.workers.mu.Lock()
		linked := len(h.workers.added)
		h.workers.mu.Unlock()

		assert.Equal(t, 3, linked, "the worker is activated after its actions are linked")
	}

	_, _ = h.open(t, link.OpenOpts{Actions: actions})

	require.Len(t, h.workers.creates, 1)
	assert.Empty(t, h.workers.creates[0].Actions)

	require.Len(t, h.workers.added, 3)
	assert.Len(t, h.workers.added[0], maxActionsPerDelta)
	assert.Len(t, h.workers.added[1], maxActionsPerDelta)
	assert.Len(t, h.workers.added[2], 500)

	linked := make([]string, 0, 2500)

	for _, chunk := range h.workers.added {
		linked = append(linked, chunk...)
	}

	assert.Equal(t, actions, linked)
	assert.Equal(t, 2, h.dispatcher.notifyCount(), "one notification for the bulk link, one for the session")
}
