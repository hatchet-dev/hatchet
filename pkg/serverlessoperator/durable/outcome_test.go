//go:build !e2e && !load && !rampup && !integration

package durable

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// TestRelayTimeoutAfterEvictionAckIsEvicted is the F10 regression: once the engine has
// acknowledged the eviction, a request timeout must not turn the outcome into a retryable
// failure, which would send a terminal event for an invocation the engine already evicted.
func TestRelayTimeoutAfterEvictionAckIsEvicted(t *testing.T) {
	setT(t)

	ch := newFakeChannel()
	seen := make(chan struct{})

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EvictionAck{
			EvictionAck: &v1.DurableTaskEvictionAckResponse{InvocationCount: 3, DurableTaskExternalId: testTaskId},
		}})

		require.NotNil(t, readResponse(t, conn).GetEvictionAck())
		close(seen)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	out := run(ctx, testParams(ep, ch))
	<-seen

	o := await(t, out)
	assert.Equal(t, KindEvicted, o.Kind, "kind=%d retry=%v close=%d error=%q", o.Kind, o.Retry, o.CloseCode, o.Error)
	assert.Equal(t, EvictionSourceEndpoint, o.EvictionSource)
	assert.Equal(t, CloseTimeout, ep.closed(t))
}

// TestRelayUnacknowledgedDoneEvictedIsProtocolViolation is the F11 regression: an endpoint
// may only report done evicted after the engine acknowledged its eviction. Without the ack
// the engine has not recorded anything, so suppressing the terminal event would leave the
// task running until the engine times it out.
func TestRelayUnacknowledgedDoneEvictedIsProtocolViolation(t *testing.T) {
	setT(t)

	ch := newFakeChannel()

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"done":{"status":"evicted"}}`)
	})

	o := await(t, run(context.Background(), testParams(ep, ch)))
	assert.Empty(t, ch.sent, "no eviction was ever requested")
	assert.Equal(t, KindFailed, o.Kind, "kind=%d retry=%v close=%d error=%q", o.Kind, o.Retry, o.CloseCode, o.Error)
	assert.True(t, o.Retry)
	assert.Equal(t, CloseForbiddenMessage, o.CloseCode)
	assert.Equal(t, CloseForbiddenMessage, ep.closed(t))
}

// TestRelayLifecycleLeaksNoGoroutines runs the relay end to end many times and checks that
// every goroutine it started is gone.
func TestRelayLifecycleLeaksNoGoroutines(t *testing.T) {
	setT(t)

	ignore := goleak.IgnoreCurrent()

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"done":{"output":"{\"ok\":true}"}}`)
	})

	for i := 0; i < 20; i++ {
		ch := newFakeChannel()
		out := Run(context.Background(), testParams(ep, ch))
		require.Equal(t, KindCompleted, out.Kind, "%+v", out)
		ep.closed(t)
	}

	ep.srv.Close()
	goleak.VerifyNone(t, ignore)
}

// TestResolveTable checks every (phase, exit) pair of the outcome table in outcome.go.
func TestResolveTable(t *testing.T) {
	const engineErr = "replay diverged"

	output := json.RawMessage(`{"ok":true}`)

	exits := map[exitKind]exit{
		exitDoneOutput:        {kind: exitDoneOutput, closeCode: CloseNormal, output: output},
		exitDoneError:         {kind: exitDoneError, closeCode: CloseNormal, msg: "boom", retry: true},
		exitDoneEvicted:       {kind: exitDoneEvicted, closeCode: CloseNormal},
		exitClosedWithoutDone: {kind: exitClosedWithoutDone, msg: "endpoint closed the websocket"},
		exitUnresponsive:      {kind: exitUnresponsive, closeCode: CloseUnresponsive, msg: "endpoint missed 2 pings"},
		exitBackpressure:      {kind: exitBackpressure, closeCode: CloseBackpressure, msg: "endpoint fell behind"},
		exitLinkFailure:       {kind: exitLinkFailure, closeCode: CloseInternalError, msg: "engine link failed"},
		exitProtocolViolation: {kind: exitProtocolViolation, closeCode: CloseInvocationMismatch, msg: "another task id"},
		exitServerEvict:       {kind: exitServerEvict, closeCode: CloseEvicted},
		exitTimeout:           {kind: exitTimeout, closeCode: CloseTimeout, msg: "timed out"},
		exitCancelled:         {kind: exitCancelled, closeCode: CloseCancelled},
		exitShutdown:          {kind: exitShutdown, closeCode: CloseShuttingDown},
	}

	type want struct {
		kind      Kind
		closeCode int
		errMsg    string
		source    string
		retry     bool
	}

	// The exit's own close code, kept in every cell but one.
	own := -1

	cells := []struct {
		name  string
		phase phase
		exit  exitKind
		want  want
	}{
		{"running/done output", phaseRunning, exitDoneOutput, want{kind: KindCompleted, closeCode: own}},
		{"running/done error", phaseRunning, exitDoneError, want{kind: KindFailed, closeCode: own, errMsg: "boom", retry: true}},
		{"running/done evicted", phaseRunning, exitDoneEvicted, want{kind: KindFailed, closeCode: CloseForbiddenMessage, errMsg: unacknowledgedEvictionMsg, retry: true}},
		{"running/closed without done", phaseRunning, exitClosedWithoutDone, want{kind: KindFailed, closeCode: own, errMsg: "endpoint closed the websocket", retry: true}},
		{"running/unresponsive", phaseRunning, exitUnresponsive, want{kind: KindFailed, closeCode: own, errMsg: "endpoint missed 2 pings", retry: true}},
		{"running/backpressure", phaseRunning, exitBackpressure, want{kind: KindFailed, closeCode: own, errMsg: "endpoint fell behind", retry: true}},
		{"running/link failure", phaseRunning, exitLinkFailure, want{kind: KindFailed, closeCode: own, errMsg: "engine link failed", retry: true}},
		{"running/protocol violation", phaseRunning, exitProtocolViolation, want{kind: KindFailed, closeCode: own, errMsg: "another task id"}},
		{"running/server evict", phaseRunning, exitServerEvict, want{kind: KindEvicted, closeCode: own, source: EvictionSourceServer}},
		{"running/timeout", phaseRunning, exitTimeout, want{kind: KindFailed, closeCode: own, errMsg: "timed out", retry: true}},
		{"running/cancelled", phaseRunning, exitCancelled, want{kind: KindCancelled, closeCode: own}},
		{"running/shutdown", phaseRunning, exitShutdown, want{kind: KindShutdown, closeCode: own}},

		{"acked/done output", phaseEvictionAcked, exitDoneOutput, want{kind: KindEvicted, closeCode: own, source: EvictionSourceEndpoint}},
		{"acked/done error", phaseEvictionAcked, exitDoneError, want{kind: KindFailed, closeCode: own, errMsg: "boom", retry: true}},
		{"acked/done evicted", phaseEvictionAcked, exitDoneEvicted, want{kind: KindEvicted, closeCode: own, source: EvictionSourceEndpoint}},
		{"acked/closed without done", phaseEvictionAcked, exitClosedWithoutDone, want{kind: KindEvicted, closeCode: own, source: EvictionSourceEndpoint}},
		{"acked/unresponsive", phaseEvictionAcked, exitUnresponsive, want{kind: KindEvicted, closeCode: own, source: EvictionSourceEndpoint}},
		{"acked/backpressure", phaseEvictionAcked, exitBackpressure, want{kind: KindEvicted, closeCode: own, source: EvictionSourceEndpoint}},
		{"acked/link failure", phaseEvictionAcked, exitLinkFailure, want{kind: KindEvicted, closeCode: own, source: EvictionSourceEndpoint}},
		{"acked/protocol violation", phaseEvictionAcked, exitProtocolViolation, want{kind: KindEvicted, closeCode: own, source: EvictionSourceEndpoint}},
		{"acked/server evict", phaseEvictionAcked, exitServerEvict, want{kind: KindEvicted, closeCode: own, source: EvictionSourceServer}},
		{"acked/timeout", phaseEvictionAcked, exitTimeout, want{kind: KindEvicted, closeCode: own, source: EvictionSourceEndpoint}},
		{"acked/cancelled", phaseEvictionAcked, exitCancelled, want{kind: KindCancelled, closeCode: own}},
		{"acked/shutdown", phaseEvictionAcked, exitShutdown, want{kind: KindEvicted, closeCode: own, source: EvictionSourceEndpoint}},

		{"errored/done output", phaseEngineErrored, exitDoneOutput, want{kind: KindFailed, closeCode: own, errMsg: engineErr}},
		{"errored/done error", phaseEngineErrored, exitDoneError, want{kind: KindFailed, closeCode: own, errMsg: engineErr}},
		{"errored/done evicted", phaseEngineErrored, exitDoneEvicted, want{kind: KindFailed, closeCode: own, errMsg: engineErr}},
		{"errored/closed without done", phaseEngineErrored, exitClosedWithoutDone, want{kind: KindFailed, closeCode: own, errMsg: engineErr}},
		{"errored/unresponsive", phaseEngineErrored, exitUnresponsive, want{kind: KindFailed, closeCode: own, errMsg: engineErr}},
		{"errored/backpressure", phaseEngineErrored, exitBackpressure, want{kind: KindFailed, closeCode: own, errMsg: engineErr}},
		{"errored/link failure", phaseEngineErrored, exitLinkFailure, want{kind: KindFailed, closeCode: own, errMsg: engineErr}},
		{"errored/protocol violation", phaseEngineErrored, exitProtocolViolation, want{kind: KindFailed, closeCode: own, errMsg: engineErr}},
		{"errored/server evict", phaseEngineErrored, exitServerEvict, want{kind: KindEvicted, closeCode: own, source: EvictionSourceServer}},
		{"errored/timeout", phaseEngineErrored, exitTimeout, want{kind: KindFailed, closeCode: own, errMsg: engineErr}},
		{"errored/cancelled", phaseEngineErrored, exitCancelled, want{kind: KindCancelled, closeCode: own}},
		{"errored/shutdown", phaseEngineErrored, exitShutdown, want{kind: KindFailed, closeCode: own, errMsg: engineErr}},
	}

	covered := map[phase]map[exitKind]bool{}

	for _, c := range cells {
		t.Run(c.name, func(t *testing.T) {
			e := exits[c.exit]
			require.NotZero(t, e.kind, "exit %d missing from the fixture", c.exit)

			engineMsg := ""

			if c.phase == phaseEngineErrored {
				engineMsg = engineErr
			}

			got := resolve(c.phase, engineMsg, e)

			assert.Equal(t, c.want.kind, got.Kind)
			assert.Equal(t, c.want.errMsg, got.Error)
			assert.Equal(t, c.want.retry, got.Retry)
			assert.Equal(t, c.want.source, got.EvictionSource)

			wantCode := c.want.closeCode

			if wantCode == own {
				wantCode = e.closeCode
			}

			assert.Equal(t, wantCode, got.CloseCode)

			if got.Kind == KindCompleted {
				assert.JSONEq(t, string(output), string(got.Output))
			}

			if covered[c.phase] == nil {
				covered[c.phase] = map[exitKind]bool{}
			}

			covered[c.phase][c.exit] = true
		})
	}

	for _, ph := range []phase{phaseRunning, phaseEvictionAcked, phaseEngineErrored} {
		for kind := range exits {
			assert.True(t, covered[ph][kind], "phase %d exit %d has no cell", ph, kind)
		}
	}

	assert.Len(t, cells, 3*len(exits), "every (phase, exit) pair has exactly one cell")
}

// TestRelayEngineErrorThenDoneOutputStaysFailed covers the errored column on a real
// socket: an endpoint that answers an engine error with a successful done frame does not
// turn the permanent error into a completion.
func TestRelayEngineErrorThenDoneOutputStaysFailed(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"id":1,"request":{"memo":{"key":"aw=="}}}`)
		require.NotNil(t, readFrame(t, conn).GetError())
		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	ch := newFakeChannel()
	out := run(context.Background(), testParams(ep, ch))

	ch.next(t)
	ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_Error{
		Error: &v1.DurableTaskErrorResponse{
			ErrorType:    v1.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_NONDETERMINISM,
			ErrorMessage: "replay diverged",
		},
	}})

	o := await(t, out)
	assert.Equal(t, KindFailed, o.Kind)
	assert.False(t, o.Retry)
	assert.Equal(t, "replay diverged", o.Error)
	assert.Equal(t, CloseNormal, ep.closed(t))
}
