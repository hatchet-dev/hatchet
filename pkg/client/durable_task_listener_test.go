package client

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// mockDurableTaskStream simulates a bidirectional gRPC stream for testing.
type mockDurableTaskStream struct {
	recvCh  chan *v1.DurableTaskResponse
	recvErr error
	sendCh  chan *v1.DurableTaskRequest
	ctx     context.Context

	v1.V1Dispatcher_DurableTaskClient
}

func newMockStream(ctx context.Context) *mockDurableTaskStream {
	return &mockDurableTaskStream{
		recvCh: make(chan *v1.DurableTaskResponse, 10),
		sendCh: make(chan *v1.DurableTaskRequest, 10),
		ctx:    ctx,
	}
}

func (m *mockDurableTaskStream) Send(req *v1.DurableTaskRequest) error {
	select {
	case m.sendCh <- req:
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
}

func (m *mockDurableTaskStream) Recv() (*v1.DurableTaskResponse, error) {
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	select {
	case resp, ok := <-m.recvCh:
		if !ok {
			return nil, io.EOF
		}
		return resp, nil
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

type testHarness struct {
	listener  *DurableTaskListener
	streams   []*mockDurableTaskStream
	mu        sync.Mutex
	callCount int
}

func newTestHarness() *testHarness {
	l := zerolog.Nop()
	h := &testHarness{}

	listener := NewDurableTaskListener(
		"test-worker",
		func(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error) {
			h.mu.Lock()
			idx := h.callCount
			if idx >= len(h.streams) {
				idx = len(h.streams) - 1
			}
			h.callCount++
			h.mu.Unlock()

			if idx < 0 {
				return nil, io.EOF
			}
			return h.streams[idx], nil
		},
		&l,
		WithReconnectInterval(10*time.Millisecond),
	)

	h.listener = listener
	return h
}

func (h *testHarness) addEOFStream(ctx context.Context) *mockDurableTaskStream {
	s := newMockStream(ctx)
	close(s.recvCh)
	h.streams = append(h.streams, s)
	return s
}

func (h *testHarness) addHangingStream(ctx context.Context) *mockDurableTaskStream {
	s := newMockStream(ctx)
	h.streams = append(h.streams, s)
	return s
}

func (h *testHarness) addErrorStream(ctx context.Context, err error) *mockDurableTaskStream {
	s := newMockStream(ctx)
	s.recvErr = err
	h.streams = append(h.streams, s)
	return s
}

func (h *testHarness) getCallCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.callCount
}

// --- Reconnection on stream EOF ---

func TestOpensNewStreamAfterEOF(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newTestHarness()
	h.addEOFStream(ctx)
	h.addHangingStream(ctx)

	h.listener.Start(ctx)
	defer h.listener.Stop()

	time.Sleep(150 * time.Millisecond)
	assert.GreaterOrEqual(t, h.getCallCount(), 2)
}

func TestMultipleEOFReconnects(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newTestHarness()
	for i := 0; i < 3; i++ {
		h.addEOFStream(ctx)
	}
	h.addHangingStream(ctx)

	h.listener.Start(ctx)
	defer h.listener.Stop()

	time.Sleep(300 * time.Millisecond)
	assert.GreaterOrEqual(t, h.getCallCount(), 4)
}

// --- Reconnection on gRPC error ---

func TestReconnectsOnUnavailable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newTestHarness()
	h.addErrorStream(ctx, status.Error(codes.Unavailable, "server unavailable"))
	h.addHangingStream(ctx)

	h.listener.Start(ctx)
	defer h.listener.Stop()

	time.Sleep(150 * time.Millisecond)
	assert.GreaterOrEqual(t, h.getCallCount(), 2)
}

func TestReconnectsOnInternalError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newTestHarness()
	h.addErrorStream(ctx, status.Error(codes.Internal, "internal"))
	h.addHangingStream(ctx)

	h.listener.Start(ctx)
	defer h.listener.Stop()

	time.Sleep(150 * time.Millisecond)
	assert.GreaterOrEqual(t, h.getCallCount(), 2)
}

// --- Does NOT reconnect on CANCELLED ---

func TestBreaksOutOnGRPCCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newTestHarness()
	h.addErrorStream(ctx, status.Error(codes.Canceled, "cancelled"))
	h.addHangingStream(ctx)

	h.listener.Start(ctx)
	defer h.listener.Stop()

	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, 1, h.getCallCount())
}

// --- Does NOT reconnect after stop ---

func TestNoReconnectAfterStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newTestHarness()
	h.addHangingStream(ctx)

	h.listener.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	h.listener.Stop()
	time.Sleep(150 * time.Millisecond)

	assert.Equal(t, 1, h.getCallCount())
}

// --- Pending acks cleared on disconnect ---

func TestFailPendingAcksClearsEventAcks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newTestHarness()
	h.addHangingStream(ctx)

	h.listener.Start(ctx)
	defer h.listener.Stop()
	time.Sleep(50 * time.Millisecond)

	ackKey := PendingAckKey{TaskID: "task1", SignalKey: 1}
	ch := h.listener.AddPendingEventAck(ackKey)

	h.listener.failPendingAcks(io.ErrUnexpectedEOF)

	select {
	case res := <-ch:
		require.Error(t, res.Err)
	case <-time.After(time.Second):
		t.Fatal("expected pending event ack to be failed")
	}

	assert.Equal(t, 0, h.listener.PendingEventAckCount())
}

// --- Pending callbacks survive disconnect ---

func TestPendingCallbacksSurviveDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newTestHarness()
	h.addEOFStream(ctx)
	h.addHangingStream(ctx)

	h.listener.Start(ctx)
	defer h.listener.Stop()

	cbKey := PendingCallbackKey{TaskID: "task1", SignalKey: 1, NodeID: 0, BranchID: 1}
	ch := h.listener.AddPendingCallback(cbKey)

	time.Sleep(150 * time.Millisecond)

	// Channel should NOT have a value - callbacks survive disconnect
	select {
	case <-ch:
		t.Fatal("pending callbacks should survive disconnect")
	default:
	}

	assert.Equal(t, 1, h.listener.PendingCallbackCount())
}

// --- Pending eviction acks cleared on disconnect ---

func TestFailPendingEvictionAcksOnDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newTestHarness()
	h.addEOFStream(ctx)
	h.addHangingStream(ctx)

	h.listener.Start(ctx)
	defer h.listener.Stop()

	ackKey := PendingAckKey{TaskID: "task1", SignalKey: 1}
	ch := h.listener.AddPendingEvictionAck(ackKey)

	time.Sleep(150 * time.Millisecond)

	select {
	case err := <-ch:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("expected pending eviction ack to be failed on disconnect")
	}

	assert.Equal(t, 0, h.listener.PendingEvictionAckCount())
}

// --- Still running after reconnect ---

func TestStillRunningAfterReconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newTestHarness()
	h.addEOFStream(ctx)
	h.addHangingStream(ctx)

	h.listener.Start(ctx)
	defer h.listener.Stop()

	time.Sleep(150 * time.Millisecond)
	assert.True(t, h.listener.IsRunning())
}

func TestNewStreamAfterReconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newTestHarness()
	h.addEOFStream(ctx)
	h.addHangingStream(ctx)

	h.listener.Start(ctx)
	defer h.listener.Stop()

	time.Sleep(150 * time.Millisecond)
	assert.GreaterOrEqual(t, h.listener.StreamSeq(), 2)
}

func TestDispatchErrorDeliversNonDeterminismError(t *testing.T) {
	l := zerolog.Nop()
	listener := NewDurableTaskListener(
		"test-worker",
		func(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error) { return nil, io.EOF },
		&l,
	)

	ackKey := PendingAckKey{TaskID: "task-x", SignalKey: 7}
	ackCh := listener.AddPendingEventAck(ackKey)

	listener.dispatchResponse(&v1.DurableTaskResponse{
		Message: &v1.DurableTaskResponse_Error{
			Error: &v1.DurableTaskErrorResponse{
				Ref: &v1.DurableEventLogEntryRef{
					DurableTaskExternalId: "task-x",
					InvocationCount:       7,
					NodeId:                3,
				},
				ErrorType:    v1.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_NONDETERMINISM,
				ErrorMessage: "branch drift",
			},
		},
	})

	select {
	case res := <-ackCh:
		require.Error(t, res.Err)
		nde, ok := res.Err.(*NonDeterminismError)
		require.True(t, ok, "expected NonDeterminismError, got %T", res.Err)
		assert.Equal(t, "task-x", nde.TaskExternalID)
		assert.Equal(t, int32(7), nde.InvocationCount)
		assert.Equal(t, int64(3), nde.NodeID)
	case <-time.After(time.Second):
		t.Fatal("expected error to be dispatched to pending event ack")
	}
}

func TestServerEvictInvokesCallbackAndCleansState(t *testing.T) {
	l := zerolog.Nop()
	listener := NewDurableTaskListener(
		"test-worker",
		func(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error) { return nil, io.EOF },
		&l,
	)

	var mu sync.Mutex
	var calls []string

	listener.SetServerEvictCallback(func(taskID string, invocation int32, reason string) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, taskID)
	})

	staleKey := PendingCallbackKey{TaskID: "task-y", SignalKey: 1, BranchID: 0, NodeID: 1}
	cbCh := listener.AddPendingCallback(staleKey)

	ackKey := PendingAckKey{TaskID: "task-y", SignalKey: 1}
	ackCh := listener.AddPendingEventAck(ackKey)

	listener.dispatchResponse(&v1.DurableTaskResponse{
		Message: &v1.DurableTaskResponse_ServerEvict{
			ServerEvict: &v1.DurableTaskServerEvictNotice{
				DurableTaskExternalId: "task-y",
				InvocationCount:       1,
				Reason:                "capacity",
			},
		},
	})

	mu.Lock()
	gotCalls := append([]string{}, calls...)
	mu.Unlock()
	require.Equal(t, []string{"task-y"}, gotCalls)

	select {
	case res := <-cbCh:
		require.Error(t, res.Err)
	case <-time.After(time.Second):
		t.Fatal("expected stale pending callback to be cancelled")
	}

	select {
	case res := <-ackCh:
		require.Error(t, res.Err)
	case <-time.After(time.Second):
		t.Fatal("expected stale pending event ack to be cancelled")
	}

	assert.Equal(t, 0, listener.PendingCallbackCount())
	assert.Equal(t, 0, listener.PendingEventAckCount())
}

func TestSendEvictionRequestTimesOut(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := zerolog.Nop()
	listener := NewDurableTaskListener(
		"test-worker",
		func(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error) {
			s := newMockStream(ctx)
			return s, nil
		},
		&l,
		WithReconnectInterval(10*time.Millisecond),
		WithEvictionAckTimeout(50*time.Millisecond),
	)

	listener.Start(ctx)
	defer listener.Stop()

	err := listener.SendEvictionRequest(ctx, "task-z", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	assert.Equal(t, 0, listener.PendingEvictionAckCount())
}

func TestBufferedCompletionDeliveredToLateConsumer(t *testing.T) {
	l := zerolog.Nop()
	listener := NewDurableTaskListener(
		"test-worker",
		func(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error) { return nil, io.EOF },
		&l,
	)

	listener.dispatchResponse(&v1.DurableTaskResponse{
		Message: &v1.DurableTaskResponse_EntryCompleted{
			EntryCompleted: &v1.DurableTaskEventLogEntryCompletedResponse{
				Ref: &v1.DurableEventLogEntryRef{
					DurableTaskExternalId: "task-q",
					InvocationCount:       2,
					BranchId:              1,
					NodeId:                4,
				},
				Payload: []byte("hello"),
			},
		},
	})

	assert.Equal(t, 1, listener.BufferedCompletionCount())

	key := PendingCallbackKey{TaskID: "task-q", SignalKey: 2, BranchID: 1, NodeID: 4}
	ch := listener.AddPendingCallback(key)

	select {
	case res := <-ch:
		require.NoError(t, res.Err)
		require.NotNil(t, res.Resp)
	case <-time.After(time.Second):
		t.Fatal("expected buffered completion to be delivered")
	}

	assert.Equal(t, 0, listener.BufferedCompletionCount())
}
