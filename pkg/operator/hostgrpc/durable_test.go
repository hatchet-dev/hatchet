//go:build !e2e && !load && !rampup && !integration

package hostgrpc

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

const eventually = 3 * time.Second

// fakeDurableStream is the engine side of one DurableTask stream: requests the listener
// sends land on sent (listener-internal register_worker and worker_status are dropped), and
// responses pushed on recv are what the listener receives.
type fakeDurableStream struct {
	v1.V1Dispatcher_DurableTaskClient
	sent chan *v1.DurableTaskRequest
	recv chan *v1.DurableTaskResponse
	done chan struct{}
	once sync.Once
}

func newFakeDurableStream() *fakeDurableStream {
	return &fakeDurableStream{
		sent: make(chan *v1.DurableTaskRequest, 64),
		recv: make(chan *v1.DurableTaskResponse, 64),
		done: make(chan struct{}),
	}
}

func (f *fakeDurableStream) Send(req *v1.DurableTaskRequest) error {
	switch req.Message.(type) {
	case *v1.DurableTaskRequest_RegisterWorker, *v1.DurableTaskRequest_WorkerStatus:
		return nil
	}

	select {
	case f.sent <- req:
		return nil
	case <-f.done:
		return io.EOF
	}
}

func (f *fakeDurableStream) Recv() (*v1.DurableTaskResponse, error) {
	select {
	case resp := <-f.recv:
		return resp, nil
	case <-f.done:
		return nil, io.EOF
	}
}

func (f *fakeDurableStream) end() {
	f.once.Do(func() { close(f.done) })
}

// next returns the next request the listener sent, or fails the test.
func (f *fakeDurableStream) next(t *testing.T) *v1.DurableTaskRequest {
	t.Helper()

	select {
	case req := <-f.sent:
		return req
	case <-time.After(eventually):
		t.Fatal("listener sent nothing")
		return nil
	}
}

func recvOne(t *testing.T, ch operator.DurableChannel) *v1.DurableTaskResponse {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), eventually)
	defer cancel()

	resp, err := ch.Recv(ctx)
	require.NoError(t, err, "Recv returned nothing")

	return resp
}

func recvErr(t *testing.T, ch operator.DurableChannel) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), eventually)
	defer cancel()

	_, err := ch.Recv(ctx)
	require.Error(t, err)
	require.NotErrorIs(t, err, context.DeadlineExceeded, "Recv returned nothing")

	return err
}

// openTestSession builds a host session over a fake client session, without a host.
func openTestSession(t *testing.T) (*session, *fakeSession) {
	t.Helper()

	fs := newFakeSession(uuid.NewString(), uuid.NewString())
	reg, err := parseRegistration(fs.Registration())
	require.NoError(t, err)

	nop := newNopLogger()
	s := newSession(fs, reg, &recordingHandler{}, nop)
	require.NoError(t, s.startDelivery())

	t.Cleanup(func() {
		_ = s.Close(context.Background())
		fs.stream.end()
	})

	return s, fs
}

func openTestChannel(t *testing.T) (*session, *fakeSession, operator.DurableChannel) {
	t.Helper()

	s, fs := openTestSession(t)

	ch, err := s.OpenDurable(context.Background(), taskId1, 2)
	require.NoError(t, err)

	return s, fs, ch
}

var (
	taskId1 = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	taskId2 = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	taskId3 = uuid.MustParse("00000000-0000-0000-0000-000000000003")
)

func ref(task uuid.UUID, inv int32, branch, node int64) *v1.DurableEventLogEntryRef {
	return &v1.DurableEventLogEntryRef{DurableTaskExternalId: task.String(), InvocationCount: inv, BranchId: branch, NodeId: node}
}

func TestDurableChannelMemo(t *testing.T) {
	_, fs, ch := openTestChannel(t)
	ctx := context.Background()

	// Ids are stamped even when the operator left them empty.
	require.NoError(t, ch.Send(ctx, &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{
		Memo: &v1.DurableTaskMemoRequest{Key: []byte("k")},
	}}))

	sent := fs.stream.next(t).GetMemo()
	require.NotNil(t, sent)
	assert.Equal(t, taskId1.String(), sent.DurableTaskExternalId)
	assert.Equal(t, int32(2), sent.InvocationCount)

	fs.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
		MemoAck: &v1.DurableTaskEventMemoAckResponse{Ref: ref(taskId1, 2, 0, 7), MemoAlreadyExisted: true, MemoResultPayload: []byte(`{"v":1}`)},
	}}

	ack := recvOne(t, ch).GetMemoAck()
	require.NotNil(t, ack)
	assert.True(t, ack.MemoAlreadyExisted)
	assert.Equal(t, int64(7), ack.Ref.NodeId)

	// complete_memo is fire-and-forget and does not count as in flight.
	require.NoError(t, ch.Send(ctx, &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_CompleteMemo{
		CompleteMemo: &v1.DurableTaskCompleteMemoRequest{Ref: ref(uuid.Nil, 0, 0, 7), Payload: []byte(`{"v":2}`)},
	}}))

	complete := fs.stream.next(t).GetCompleteMemo()
	require.NotNil(t, complete)
	assert.Equal(t, taskId1.String(), complete.Ref.DurableTaskExternalId)
	assert.Equal(t, int32(2), complete.Ref.InvocationCount)

	// worker_status is the listener's over gRPC and is dropped
	require.NoError(t, ch.Send(ctx, &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_WorkerStatus{
		WorkerStatus: &v1.DurableTaskWorkerStatusRequest{},
	}}))
}

func TestDurableChannelWaitForDeliversEntryCompleted(t *testing.T) {
	s, fs, ch := openTestChannel(t)
	ctx := context.Background()

	require.NoError(t, ch.Send(ctx, &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_WaitFor{
		WaitFor: &v1.DurableTaskWaitForRequest{},
	}}))

	require.NotNil(t, fs.stream.next(t).GetWaitFor())

	fs.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_WaitForAck{
		WaitForAck: &v1.DurableTaskEventWaitForAckResponse{Ref: ref(taskId1, 2, 1, 3)},
	}}

	require.NotNil(t, recvOne(t, ch).GetWaitForAck())

	// The callback is registered under (task, invocation, branch, node) like the SDK's
	// WaitForCallback, and the worker status advertises it.
	require.Eventually(t, func() bool { return s.hub.listener.PendingCallbackCount() == 1 }, eventually, 10*time.Millisecond)

	fs.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EntryCompleted{
		EntryCompleted: &v1.DurableTaskEventLogEntryCompletedResponse{Ref: ref(taskId1, 2, 1, 3), Payload: []byte(`{"slept":true}`)},
	}}

	completed := recvOne(t, ch).GetEntryCompleted()
	require.NotNil(t, completed)
	assert.Equal(t, `{"slept":true}`, string(completed.Payload))
}

func TestDurableChannelTriggerRunsDeliversChildCompletions(t *testing.T) {
	s, fs, ch := openTestChannel(t)
	ctx := context.Background()

	require.NoError(t, ch.Send(ctx, &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_TriggerRuns{
		TriggerRuns: &v1.DurableTaskTriggerRunsRequest{TriggerOpts: []*v1.TriggerWorkflowRequest{{Name: "child"}}},
	}}))

	require.NotNil(t, fs.stream.next(t).GetTriggerRuns())

	fs.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_TriggerRunsAck{
		TriggerRunsAck: &v1.DurableTaskEventTriggerRunsAckResponse{
			DurableTaskExternalId: taskId1.String(),
			InvocationCount:       2,
			RunEntries:            []*v1.DurableTaskRunAckEntry{{NodeId: 5, BranchId: 0, WorkflowRunExternalId: "run-a"}},
		},
	}}

	require.NotNil(t, recvOne(t, ch).GetTriggerRunsAck())
	require.Eventually(t, func() bool { return s.hub.listener.PendingCallbackCount() == 1 }, eventually, 10*time.Millisecond)

	fs.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EntryCompleted{
		EntryCompleted: &v1.DurableTaskEventLogEntryCompletedResponse{Ref: ref(taskId1, 2, 0, 5), Payload: []byte(`{"child":1}`)},
	}}

	completed := recvOne(t, ch).GetEntryCompleted()
	require.NotNil(t, completed)
	assert.Equal(t, int64(5), completed.Ref.NodeId)
}

func TestDurableChannelEviction(t *testing.T) {
	_, fs, ch := openTestChannel(t)

	require.NoError(t, ch.Send(context.Background(), &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_EvictInvocation{
		EvictInvocation: &v1.DurableTaskEvictInvocationRequest{},
	}}))

	evict := fs.stream.next(t).GetEvictInvocation()
	require.NotNil(t, evict)
	assert.Equal(t, taskId1.String(), evict.DurableTaskExternalId)
	assert.Equal(t, int32(2), evict.InvocationCount)

	fs.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EvictionAck{
		EvictionAck: &v1.DurableTaskEvictionAckResponse{DurableTaskExternalId: taskId1.String(), InvocationCount: 2},
	}}

	ack := recvOne(t, ch).GetEvictionAck()
	require.NotNil(t, ack)
	assert.Equal(t, int32(2), ack.InvocationCount)
}

func TestDurableChannelServerEvict(t *testing.T) {
	_, fs, ch := openTestChannel(t)

	// A memo is in flight when the engine supersedes the invocation: the listener fails the
	// ack during cleanup, which the eviction explains, so Recv yields server_evict, not an
	// error.
	require.NoError(t, ch.Send(context.Background(), &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{
		Memo: &v1.DurableTaskMemoRequest{Key: []byte("k")},
	}}))
	fs.stream.next(t)

	fs.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_ServerEvict{
		ServerEvict: &v1.DurableTaskServerEvictNotice{DurableTaskExternalId: taskId1.String(), InvocationCount: 2, Reason: "superseded"},
	}}

	notice := recvOne(t, ch).GetServerEvict()
	require.NotNil(t, notice)
	assert.Equal(t, "superseded", notice.Reason)
}

func TestDurableChannelNonDeterminismBecomesErrorResponse(t *testing.T) {
	_, fs, ch := openTestChannel(t)

	require.NoError(t, ch.Send(context.Background(), &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{
		Memo: &v1.DurableTaskMemoRequest{Key: []byte("k")},
	}}))
	fs.stream.next(t)

	fs.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_Error{
		Error: &v1.DurableTaskErrorResponse{
			Ref:          ref(taskId1, 2, 0, 1),
			ErrorType:    v1.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_NONDETERMINISM,
			ErrorMessage: "replay diverged",
		},
	}}

	errResp := recvOne(t, ch).GetError()
	require.NotNil(t, errResp)
	assert.Equal(t, v1.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_NONDETERMINISM, errResp.ErrorType)
	assert.Equal(t, "replay diverged", errResp.ErrorMessage)
	assert.Equal(t, int64(1), errResp.Ref.NodeId)
}

func TestDurableChannelOneInFlight(t *testing.T) {
	_, fs, ch := openTestChannel(t)
	ctx := context.Background()

	memo := func() *v1.DurableTaskRequest {
		return &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{Memo: &v1.DurableTaskMemoRequest{Key: []byte("k")}}}
	}

	require.NoError(t, ch.Send(ctx, memo()))
	assert.ErrorIs(t, ch.Send(ctx, memo()), operator.ErrRequestInFlight)
	assert.ErrorIs(t, ch.Send(ctx, &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_EvictInvocation{
		EvictInvocation: &v1.DurableTaskEvictInvocationRequest{},
	}}), operator.ErrRequestInFlight)

	fs.stream.next(t)
	fs.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
		MemoAck: &v1.DurableTaskEventMemoAckResponse{Ref: ref(taskId1, 2, 0, 1)},
	}}
	recvOne(t, ch)

	// The slot is free again once the ack arrived.
	require.NoError(t, ch.Send(ctx, memo()))
}

func TestDurableChannelRejectsSessionOwnedRequests(t *testing.T) {
	_, _, ch := openTestChannel(t)

	assert.Error(t, ch.Send(context.Background(), &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_RegisterWorker{
		RegisterWorker: &v1.DurableTaskRequestRegisterWorker{WorkerId: "x"},
	}}))
}

func TestDurableChannelTransportFailureIsAnError(t *testing.T) {
	s, fs, ch := openTestChannel(t)

	require.NoError(t, ch.Send(context.Background(), &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{
		Memo: &v1.DurableTaskMemoRequest{Key: []byte("k")},
	}}))
	fs.stream.next(t)

	// Stopping the listener fails the pending ack with no eviction to explain it.
	s.hub.listener.Stop()

	err := recvErr(t, ch)
	assert.Contains(t, err.Error(), "listener stopped")
}

// Recv honours the caller's context.
func TestDurableChannelRecvContext(t *testing.T) {
	_, _, ch := openTestChannel(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := ch.Recv(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestDurableChannelCloseCleansUp(t *testing.T) {
	s, fs, ch := openTestChannel(t)
	ctx := context.Background()

	require.NoError(t, ch.Send(ctx, &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_WaitFor{
		WaitFor: &v1.DurableTaskWaitForRequest{},
	}}))
	fs.stream.next(t)
	fs.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_WaitForAck{
		WaitForAck: &v1.DurableTaskEventWaitForAckResponse{Ref: ref(taskId1, 2, 0, 3)},
	}}
	recvOne(t, ch)

	listener := s.hub.listener
	require.Eventually(t, func() bool { return listener.PendingCallbackCount() == 1 }, eventually, 10*time.Millisecond)

	require.NoError(t, ch.Close())
	require.NoError(t, ch.Close(), "Close is idempotent")

	assert.Equal(t, 0, listener.PendingCallbackCount())
	assert.Equal(t, 0, listener.PendingEventAckCount())
	assert.ErrorIs(t, recvErr(t, ch), operator.ErrChannelClosed)
	assert.ErrorIs(t, ch.Send(ctx, &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{
		Memo: &v1.DurableTaskMemoRequest{Key: []byte("k")},
	}}), operator.ErrChannelClosed)

	// The same invocation can be reopened after Close; a duplicate open is refused.
	again, err := s.OpenDurable(ctx, taskId1, 2)
	require.NoError(t, err)

	_, err = s.OpenDurable(ctx, taskId1, 2)
	assert.Error(t, err)

	require.NoError(t, again.Close())

	// Closing the session closes open channels and stops the listener.
	third, err := s.OpenDurable(ctx, taskId2, 0)
	require.NoError(t, err)
	require.NoError(t, s.Close(ctx))
	assert.ErrorIs(t, recvErr(t, third), operator.ErrChannelClosed)
	assert.True(t, fs.isClosed())
	assert.False(t, listener.IsRunning())

	_, err = s.OpenDurable(ctx, taskId3, 0)
	assert.ErrorIs(t, err, operator.ErrSessionClosed)
}

// A server-evict notice reaches the named task's channels at or below the named invocation
// and no other task's, through the hub's per-task index.
func TestOnServerEvictReachesOnlyTheNamedTask(t *testing.T) {
	fs := newFakeSession(uuid.NewString(), uuid.NewString())
	hub := newDurableHubOver(newDurableTaskListener(fs.workerId, fs.OpenDurableTaskStream, nil))
	defer hub.closeAll()

	older, err := hub.open("task-a", 1)
	require.NoError(t, err)
	current, err := hub.open("task-a", 2)
	require.NoError(t, err)
	newer, err := hub.open("task-a", 3)
	require.NoError(t, err)
	other, err := hub.open("task-b", 1)
	require.NoError(t, err)

	hub.onServerEvict("task-a", 2, "superseded")

	for _, ch := range []*durableChannel{older, current} {
		notice := recvOne(t, ch).GetServerEvict()
		require.NotNil(t, notice)
		assert.Equal(t, "task-a", notice.GetDurableTaskExternalId())
		assert.Equal(t, int32(2), notice.GetInvocationCount())
		assert.Equal(t, "superseded", notice.GetReason())
	}

	for _, ch := range []*durableChannel{newer, other} {
		select {
		case item := <-ch.queue:
			t.Fatalf("channel for task %s invocation %d received %v", ch.taskId, ch.invocation, item.resp)
		case <-time.After(50 * time.Millisecond):
		}
	}

	// closing every channel of a task drops the task from the index
	require.NoError(t, older.Close())
	require.NoError(t, current.Close())
	require.NoError(t, newer.Close())

	hub.mu.Lock()
	_, stillIndexed := hub.channels["task-a"]
	assert.Len(t, hub.channels["task-b"], 1)
	hub.mu.Unlock()
	assert.False(t, stillIndexed)

	// a notice for a task with no channels is a no-op
	hub.onServerEvict("task-a", 5, "gone")
}

// completeMemo is a fire-and-forget request, the kind that fills the shared listener's queue
// fastest when the engine is unreachable.
func completeMemo() *v1.DurableTaskRequest {
	return &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_CompleteMemo{
		CompleteMemo: &v1.DurableTaskCompleteMemoRequest{Ref: ref(taskId1, 1, 0, 1)},
	}}
}

// A Send blocked on a full shared request queue must return once the invocation's channel
// is closed, so a timed-out or cancelled invocation can tear down and shutdown can complete
// while the engine is unreachable; a Send whose context ends returns the same way.
func TestFullQueueSendUnblocksOnClose(t *testing.T) {
	s, _ := openTestSession(t)

	ch, err := s.OpenDurable(context.Background(), taskId1, 1)
	require.NoError(t, err)

	// Nobody reads the fake stream, so requests back up in the stream buffer, the shared
	// listener's queue and the hub's own queue, until a Send blocks.
	blocked := make(chan error, 1)
	sent := 0

	for {
		done := make(chan error, 1)

		go func() { done <- ch.Send(context.Background(), completeMemo()) }()

		select {
		case err := <-done:
			require.NoError(t, err)
			sent++

			if sent > 10000 {
				t.Fatal("sends never blocked; the queue is unbounded")
			}

			continue
		case <-time.After(100 * time.Millisecond):
			go func() { blocked <- <-done }()
		}

		break
	}

	t.Logf("%d sends queued before one blocked", sent)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	assert.ErrorIs(t, ch.Send(ctx, completeMemo()), context.DeadlineExceeded, "a Send whose context ends gives up")

	require.NoError(t, ch.Close())

	select {
	case err := <-blocked:
		assert.ErrorIs(t, err, operator.ErrChannelClosed, "a blocked Send reports the close")
	case <-time.After(2 * time.Second):
		t.Fatal("a Send blocked on the full request queue survived the channel's Close")
	}

	// A closed channel refuses further sends at once.
	assert.ErrorIs(t, ch.Send(context.Background(), completeMemo()), operator.ErrChannelClosed)
}

// A callback registration scheduled just before an invocation closes must not outlive the
// invocation on the shared listener.
func TestLateCallbackDoesNotSurviveClose(t *testing.T) {
	s, _ := openTestSession(t)

	for i := 0; i < 100; i++ {
		ch, err := s.OpenDurable(context.Background(), uuid.New(), 1)
		require.NoError(t, err)

		c := ch.(*durableChannel)

		// The goroutine an ack schedules races the close: whichever order the scheduler
		// picks, no callback may remain.
		go c.awaitEntry(0, 1)

		require.NoError(t, ch.Close())
		c.awaitEntry(0, 2)
	}

	listener := s.hub.listener

	assert.Eventually(t, func() bool { return listener.PendingCallbackCount() == 0 }, eventually, 5*time.Millisecond,
		"closed invocations left %d late callbacks on the shared listener", listener.PendingCallbackCount())
}

// An ordinary invocation lifecycle (open, ack-bearing request, ack, close, session close)
// leaves no goroutine behind: every goroutine a channel starts is joined by its Close.
func TestDurableChannelLifecycleLeaksNoGoroutines(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	fs := newFakeSession(uuid.NewString(), uuid.NewString())
	reg, err := parseRegistration(fs.Registration())
	require.NoError(t, err)

	s := newSession(fs, reg, &recordingHandler{}, newNopLogger())
	require.NoError(t, s.startDelivery())

	for i := 0; i < 20; i++ {
		taskId := uuid.New()

		ch, err := s.OpenDurable(context.Background(), taskId, 1)
		require.NoError(t, err)

		require.NoError(t, ch.Send(context.Background(), &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{Memo: &v1.DurableTaskMemoRequest{}}}))
		fs.stream.next(t)

		fs.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
			MemoAck: &v1.DurableTaskEventMemoAckResponse{Ref: ref(taskId, 1, 0, 1)},
		}}

		require.NotNil(t, recvOne(t, ch).GetMemoAck())
		require.NoError(t, ch.Close())
	}

	require.NoError(t, s.Close(context.Background()))
	fs.stream.end()
}

func TestSessionClosedErrorsAreDistinct(t *testing.T) {
	assert.NotErrorIs(t, operator.ErrSessionClosed, operator.ErrChannelClosed)
	assert.NotEqual(t, fmt.Sprint(operator.ErrSessionClosed), fmt.Sprint(operator.ErrChannelClosed))
}
