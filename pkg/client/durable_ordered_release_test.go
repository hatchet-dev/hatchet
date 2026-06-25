package client

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func newGateListener(opts ...DurableTaskListenerOpt) *DurableTaskListener {
	l := zerolog.Nop()
	return NewDurableTaskListener(
		"test-worker",
		func(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error) { return nil, io.EOF },
		&l,
		opts...,
	)
}

func entryCompleted(taskID string, invocation int32, branchID, nodeID int64, payload string, order *int64) *v1.DurableTaskResponse {
	return &v1.DurableTaskResponse{
		Message: &v1.DurableTaskResponse_EntryCompleted{
			EntryCompleted: &v1.DurableTaskEventLogEntryCompletedResponse{
				Ref: &v1.DurableEventLogEntryRef{
					DurableTaskExternalId: taskID,
					InvocationCount:       invocation,
					BranchId:              branchID,
					NodeId:                nodeID,
				},
				Payload:        []byte(payload),
				SatisfiedOrder: order,
			},
		},
	}
}

func orderPtr(o int64) *int64 {
	return &o
}

func expectDelivered(t *testing.T, ch chan CallbackResult, payload string) {
	t.Helper()
	select {
	case res := <-ch:
		require.NoError(t, res.Err)
		require.NotNil(t, res.Resp)
		assert.Equal(t, payload, string(res.Resp.GetEntryCompleted().GetPayload()))
	default:
		t.Fatal("expected completion to be delivered")
	}
}

func expectNotDelivered(t *testing.T, ch chan CallbackResult) {
	t.Helper()
	select {
	case <-ch:
		t.Fatal("expected completion to be held by the gate")
	default:
	}
}

// Out-of-order arrival: the completion stamped with a later order is held
// until the earlier order arrives, even when its waiter is already parked.
// Mirrors the A->B / C->D scenario: C's completion (order 1) must wake its
// continuation before A's completion (order 2), regardless of arrival order.
func TestOrderedRelease_HoldsOutOfOrderCompletion(t *testing.T) {
	l := newGateListener()

	chA := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 1})
	chC := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 2})

	// A's completion was stamped second but arrives first: held.
	l.dispatchResponse(entryCompleted("task", 1, 1, 1, "a-result", orderPtr(2)))
	expectNotDelivered(t, chA)

	// C's completion (order 1) arrives: released, wakes C's continuation.
	l.dispatchResponse(entryCompleted("task", 1, 1, 2, "c-result", orderPtr(1)))
	expectDelivered(t, chC, "c-result")

	// the gate stays closed for order 2 until C's continuation parks.
	expectNotDelivered(t, chA)

	// C's continuation spawns D and parks on its result: gate opens, order 2
	// is released to A's waiter.
	chD := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 3})
	expectDelivered(t, chA, "a-result")
	expectNotDelivered(t, chD)
}

// Multi-op continuation: a woken continuation issues several durable ops
// (acks do not open the gate) before parking; the next completion is held the
// entire time.
func TestOrderedRelease_GateHeldUntilPark(t *testing.T) {
	l := newGateListener()

	ch1 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 1})
	ch2 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 2})

	l.dispatchResponse(entryCompleted("task", 1, 1, 1, "first", orderPtr(1)))
	expectDelivered(t, ch1, "first")

	// order 2 arrives while the woken continuation is mid-flight (e.g. between
	// spawn acks): held.
	l.dispatchResponse(entryCompleted("task", 1, 1, 2, "second", orderPtr(2)))
	expectNotDelivered(t, ch2)

	// ack round trips must not open the gate: pending event acks don't touch it.
	ackCh := l.AddPendingEventAck(PendingAckKey{TaskID: "task", SignalKey: 1})
	_ = ackCh
	expectNotDelivered(t, ch2)

	// the continuation parks on its next durable await: gate opens.
	_ = l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 3})
	expectDelivered(t, ch2, "second")
}

// A released completion with no parked waiter is buffered and the pump keeps
// going, so a continuation sequentially awaiting a later order is not
// deadlocked.
func TestOrderedRelease_BufferedReleaseKeepsPumping(t *testing.T) {
	l := newGateListener()

	// the only parked waiter awaits the entry satisfied at order 2 (sequential
	// code awaiting A first while C completed first).
	ch2 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 1})

	l.dispatchResponse(entryCompleted("task", 1, 1, 2, "c-result", orderPtr(1)))
	l.dispatchResponse(entryCompleted("task", 1, 1, 1, "a-result", orderPtr(2)))

	// order 1 had no waiter -> buffered; order 2 released to the waiter.
	expectDelivered(t, ch2, "a-result")
	assert.Equal(t, 1, l.BufferedCompletionCount())

	// the buffered completion is picked up on late registration.
	chLate := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 2})
	expectDelivered(t, chLate, "c-result")
}

// Re-delivered completions (orders at or below the released watermark, e.g.
// after a reconnect) bypass the gate.
func TestOrderedRelease_RedeliveryBypassesGate(t *testing.T) {
	l := newGateListener()

	ch1 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 1})
	l.dispatchResponse(entryCompleted("task", 1, 1, 1, "first", orderPtr(1)))
	expectDelivered(t, ch1, "first")

	// gate is closed (waiting for the woken continuation to park), but a
	// re-delivery of order 1 is delivered immediately.
	chRetry := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 1})
	l.dispatchResponse(entryCompleted("task", 1, 1, 1, "first", orderPtr(1)))
	expectDelivered(t, chRetry, "first")
}

// Legacy completions with no satisfied order are released immediately, even
// while the gate holds ordered completions.
func TestOrderedRelease_LegacyNilOrderReleasedImmediately(t *testing.T) {
	l := newGateListener()

	chOrdered := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 1})
	chLegacy := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 9})

	// ordered completion with a gap (order 2, order 1 missing): held.
	l.dispatchResponse(entryCompleted("task", 1, 1, 1, "ordered", orderPtr(2)))
	expectNotDelivered(t, chOrdered)

	// legacy completion: delivered immediately.
	l.dispatchResponse(entryCompleted("task", 1, 1, 9, "legacy", nil))
	expectDelivered(t, chLegacy, "legacy")
}

// Gates are scoped per invocation: an ordered completion for invocation 2
// starts a fresh sequence and is not blocked by invocation 1's gate.
func TestOrderedRelease_GatesScopedPerInvocation(t *testing.T) {
	l := newGateListener()

	chInv1 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 1})
	chInv2 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 2, BranchID: 1, NodeID: 1})

	// invocation 1 is blocked on a gap.
	l.dispatchResponse(entryCompleted("task", 1, 1, 1, "inv1", orderPtr(2)))
	expectNotDelivered(t, chInv1)

	// invocation 2's order 1 releases independently.
	l.dispatchResponse(entryCompleted("task", 2, 1, 1, "inv2", orderPtr(1)))
	expectDelivered(t, chInv2, "inv2")
}

// Gap timeout: a persistent hole in the order sequence fails the invocation's
// waiters with an OrderedReplayGapError instead of hanging forever.
func TestOrderedRelease_GapTimeoutFailsWaiters(t *testing.T) {
	l := newGateListener(WithGapTimeout(10 * time.Millisecond))

	ch := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 1})
	ackCh := l.AddPendingEventAck(PendingAckKey{TaskID: "task", SignalKey: 1})

	// order 2 arrives, order 1 never does (history diverged).
	l.dispatchResponse(entryCompleted("task", 1, 1, 1, "stranded", orderPtr(2)))
	expectNotDelivered(t, ch)

	time.Sleep(20 * time.Millisecond)
	l.sweepGates()

	select {
	case res := <-ch:
		require.Error(t, res.Err)
		var gapErr *OrderedReplayGapError
		require.True(t, errors.As(res.Err, &gapErr), "expected OrderedReplayGapError, got %T", res.Err)
		assert.Equal(t, "task", gapErr.TaskExternalID)
		assert.Equal(t, int32(1), gapErr.InvocationCount)
		assert.Equal(t, int64(1), gapErr.MissingOrder)
		assert.Equal(t, []int64{2}, gapErr.HeldOrders)
	case <-time.After(time.Second):
		t.Fatal("expected gap timeout to fail the pending callback")
	}

	select {
	case res := <-ackCh:
		require.Error(t, res.Err)
	case <-time.After(time.Second):
		t.Fatal("expected gap timeout to fail the pending event ack")
	}

	l.gateMu.Lock()
	assert.Empty(t, l.gates)
	l.gateMu.Unlock()
}

// Park timeout: a woken continuation that never parks (unrecorded blocking
// work) forces the gate open instead of stalling later completions forever.
func TestOrderedRelease_ParkTimeoutForcesGateOpen(t *testing.T) {
	l := newGateListener(WithParkTimeout(10 * time.Millisecond))

	ch1 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 1})
	ch2 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 2})

	l.dispatchResponse(entryCompleted("task", 1, 1, 1, "first", orderPtr(1)))
	expectDelivered(t, ch1, "first")

	l.dispatchResponse(entryCompleted("task", 1, 1, 2, "second", orderPtr(2)))
	expectNotDelivered(t, ch2)

	time.Sleep(20 * time.Millisecond)
	l.sweepGates()

	expectDelivered(t, ch2, "second")
}

// NotifyInvocationQuiesced (durable task fn returned) opens the gate without
// waiting for the park timeout.
func TestOrderedRelease_QuiesceOpensGate(t *testing.T) {
	l := newGateListener()

	ch1 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 1})
	ch2 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 2})

	l.dispatchResponse(entryCompleted("task", 1, 1, 1, "first", orderPtr(1)))
	expectDelivered(t, ch1, "first")

	l.dispatchResponse(entryCompleted("task", 1, 1, 2, "second", orderPtr(2)))
	expectNotDelivered(t, ch2)

	l.NotifyInvocationQuiesced("task", 1)
	expectDelivered(t, ch2, "second")
}

// CleanupTaskState drops gate state for evicted invocations.
func TestOrderedRelease_CleanupDropsGates(t *testing.T) {
	l := newGateListener()

	l.dispatchResponse(entryCompleted("task", 1, 1, 1, "stranded", orderPtr(2)))

	l.gateMu.Lock()
	assert.Len(t, l.gates, 1)
	l.gateMu.Unlock()

	l.CleanupTaskState("task", 1)

	l.gateMu.Lock()
	assert.Empty(t, l.gates)
	l.gateMu.Unlock()
}

// In-order arrival releases without any holds: contiguous orders flow through
// as each woken continuation parks.
func TestOrderedRelease_InOrderFlow(t *testing.T) {
	l := newGateListener()

	ch1 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 1})
	ch2 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 2})
	ch3 := l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 3})

	l.dispatchResponse(entryCompleted("task", 1, 1, 1, "one", orderPtr(1)))
	expectDelivered(t, ch1, "one")

	l.dispatchResponse(entryCompleted("task", 1, 1, 2, "two", orderPtr(2)))
	expectNotDelivered(t, ch2)

	// continuation 1 parks -> two released; continuation 2 parks -> three released.
	_ = l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 4})
	expectDelivered(t, ch2, "two")

	l.dispatchResponse(entryCompleted("task", 1, 1, 3, "three", orderPtr(3)))
	expectNotDelivered(t, ch3)

	_ = l.AddPendingCallback(PendingCallbackKey{TaskID: "task", SignalKey: 1, BranchID: 1, NodeID: 5})
	expectDelivered(t, ch3, "three")
}
