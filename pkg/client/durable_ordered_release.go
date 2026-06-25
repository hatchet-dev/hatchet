package client

import (
	"fmt"
	"time"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

const (
	defaultParkTimeout = 5 * time.Second
	defaultGapTimeout  = 60 * time.Second
	gateSweepInterval  = 500 * time.Millisecond
)

// OrderedReplayGapError is returned to waiters when a gap in the satisfied
// order sequence persists beyond the gap timeout. This indicates the recorded
// history diverged from the current code (e.g. a deploy removed a durable
// await mid-flight), since a recorded completion that should fill the gap was
// never re-awaited.
type OrderedReplayGapError struct {
	TaskExternalID  string
	HeldOrders      []int64
	MissingOrder    int64
	InvocationCount int32
}

func (e *OrderedReplayGapError) Error() string {
	return fmt.Sprintf(
		"non-determinism detected for task %s (invocation %d): completion with satisfied order %d was never delivered while later completions %v arrived; the recorded history likely diverged from the current code",
		e.TaskExternalID, e.InvocationCount, e.MissingOrder, e.HeldOrders,
	)
}

// invocationGate serializes the release of ordered EntryCompleted responses for
// a single durable task invocation. Completions are released to user code in
// satisfied_order, and after a release wakes a parked continuation, further
// releases are held until that continuation parks again (registers its next
// awaited entry), the runtime signals quiescence, or the park timeout elapses.
type invocationGate struct {
	wakeSince time.Time
	gapSince  time.Time
	held      map[int64]*v1.DurableTaskResponse
	released  int64
	wakes     int
}

func (l *DurableTaskListener) gateFor(key PendingAckKey) *invocationGate {
	g, ok := l.gates[key]
	if !ok {
		g = &invocationGate{held: make(map[int64]*v1.DurableTaskResponse)}
		l.gates[key] = g
	}
	return g
}

// handleOrderedEntryCompleted routes an EntryCompleted response carrying a
// satisfied order through the invocation's gate. Must NOT be called while
// holding gateMu.
func (l *DurableTaskListener) handleOrderedEntryCompleted(key PendingAckKey, order int64, resp *v1.DurableTaskResponse) {
	l.gateMu.Lock()
	g := l.gateFor(key)

	if order <= g.released {
		// re-delivery of an already-released completion (e.g. after reconnect):
		// bypass the gate.
		l.gateMu.Unlock()
		l.deliverCompletion(resp)
		return
	}

	g.held[order] = resp
	l.pumpLocked(key, g)
	l.gateMu.Unlock()
}

// pumpLocked releases contiguously ordered completions while the gate is open.
// Callers must hold gateMu.
func (l *DurableTaskListener) pumpLocked(key PendingAckKey, g *invocationGate) {
	for g.wakes == 0 {
		resp, ok := g.held[g.released+1]
		if !ok {
			break
		}

		delete(g.held, g.released+1)
		g.released++

		if l.deliverCompletion(resp) {
			// the release woke a parked continuation: hold further releases
			// until it parks again.
			g.wakes++
			g.wakeSince = time.Now()
		}
		// if nobody was waiting, the completion was buffered for a continuation
		// that is still running; keep pumping so a parked continuation awaiting
		// a later order is not deadlocked.
	}

	if len(g.held) > 0 && g.wakes == 0 {
		if g.gapSince.IsZero() {
			g.gapSince = time.Now()
		}
	} else {
		g.gapSince = time.Time{}
	}
}

// deliverCompletion hands an EntryCompleted response to a registered waiter,
// or buffers it for late registration. Returns true if a parked continuation
// was woken.
func (l *DurableTaskListener) deliverCompletion(resp *v1.DurableTaskResponse) bool {
	completed := resp.GetEntryCompleted()
	ref := completed.GetRef()
	key := PendingCallbackKey{
		TaskID:    ref.GetDurableTaskExternalId(),
		SignalKey: int64(ref.GetInvocationCount()),
		BranchID:  ref.GetBranchId(),
		NodeID:    ref.GetNodeId(),
	}

	l.pendingCallbacksMu.Lock()
	ch, ok := l.pendingCallbacks[key]
	if ok {
		delete(l.pendingCallbacks, key)
	}
	l.pendingCallbacksMu.Unlock()

	if ok {
		select {
		case ch <- CallbackResult{Resp: resp}:
		default:
		}
		return true
	}

	l.bufferedCompletionsMu.Lock()
	l.bufferedCompletions[key] = resp
	l.bufferedCompletionsMu.Unlock()

	return false
}

// notifyParked records that a continuation of the given invocation parked
// (registered its next awaited entry without a buffered result), opening the
// gate for the next ordered release.
func (l *DurableTaskListener) notifyParked(key PendingAckKey) {
	l.gateMu.Lock()
	defer l.gateMu.Unlock()

	g, ok := l.gates[key]
	if !ok {
		return
	}

	if g.wakes > 0 {
		g.wakes--
	}

	l.pumpLocked(key, g)
}

// NotifyInvocationQuiesced signals that the durable task function for the given
// invocation returned (or otherwise has no running continuations), releasing
// any gate held on its behalf.
func (l *DurableTaskListener) NotifyInvocationQuiesced(taskExternalID string, invocationCount int32) {
	key := PendingAckKey{TaskID: taskExternalID, SignalKey: int64(invocationCount)}

	l.gateMu.Lock()
	defer l.gateMu.Unlock()

	g, ok := l.gates[key]
	if !ok {
		return
	}

	g.wakes = 0
	l.pumpLocked(key, g)
}

// sweepGates enforces the park and gap timeouts. Park timeout: a woken
// continuation that never parks (e.g. performed unrecorded blocking work or
// returned without the runtime quiesce hook) forces the gate open with a loud
// warning. Gap timeout: a persistent hole in the order sequence while later
// completions are held fails the invocation's waiters with an
// OrderedReplayGapError instead of hanging.
func (l *DurableTaskListener) sweepGates() {
	type gapFailure struct {
		err *OrderedReplayGapError
		key PendingAckKey
	}

	var failures []gapFailure

	l.gateMu.Lock()
	now := time.Now()

	for key, g := range l.gates {
		if g.wakes > 0 && now.Sub(g.wakeSince) > l.parkTimeout {
			l.l.Warn().
				Str("task_id", key.TaskID).
				Int64("invocation_count", key.SignalKey).
				Msgf("DurableTaskListener: continuation did not park within %s after a gated release; forcing the completion gate open. durable task code should not perform unrecorded blocking work between durable operations", l.parkTimeout)
			g.wakes = 0
			l.pumpLocked(key, g)
		}

		if len(g.held) > 0 && g.wakes == 0 && !g.gapSince.IsZero() && now.Sub(g.gapSince) > l.gapTimeout {
			heldOrders := make([]int64, 0, len(g.held))
			for o := range g.held {
				heldOrders = append(heldOrders, o)
			}

			failures = append(failures, gapFailure{
				key: key,
				err: &OrderedReplayGapError{
					TaskExternalID:  key.TaskID,
					InvocationCount: int32(key.SignalKey), // nolint:gosec
					MissingOrder:    g.released + 1,
					HeldOrders:      heldOrders,
				},
			})

			delete(l.gates, key)
		}
	}
	l.gateMu.Unlock()

	for _, f := range failures {
		l.l.Error().
			Str("task_id", f.key.TaskID).
			Int64("invocation_count", f.key.SignalKey).
			Msg(f.err.Error())

		l.failInvocationWaiters(f.key, f.err)
	}
}

// failInvocationWaiters delivers an error to every pending callback and event
// ack belonging to the given invocation.
func (l *DurableTaskListener) failInvocationWaiters(key PendingAckKey, err error) {
	l.pendingCallbacksMu.Lock()
	for k, ch := range l.pendingCallbacks {
		if k.TaskID == key.TaskID && k.SignalKey == key.SignalKey {
			delete(l.pendingCallbacks, k)
			select {
			case ch <- CallbackResult{Err: err}:
			default:
			}
		}
	}
	l.pendingCallbacksMu.Unlock()

	l.pendingEventAcksMu.Lock()
	if ch, ok := l.pendingEventAcks[key]; ok {
		delete(l.pendingEventAcks, key)
		select {
		case ch <- EventAckResult{Err: err}:
		default:
		}
	}
	l.pendingEventAcksMu.Unlock()
}

func (l *DurableTaskListener) runGateSweeper(stop <-chan struct{}) {
	ticker := time.NewTicker(gateSweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			l.sweepGates()
		}
	}
}
