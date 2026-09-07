//go:build !e2e && !load && !rampup && !integration

package grpclink

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

const eventually = 3 * time.Second

// fakeDurableStream is the engine side of one DurableTask stream: requests the listener
// sends land on sent (link-internal register_worker and worker_status are dropped), and
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

func recvOne(t *testing.T, ch link.DurableChannel) *v1.DurableTaskResponse {
	t.Helper()

	type result struct {
		resp *v1.DurableTaskResponse
		err  error
	}

	out := make(chan result, 1)

	go func() {
		resp, err := ch.Recv()
		out <- result{resp, err}
	}()

	select {
	case r := <-out:
		require.NoError(t, r.err)
		return r.resp
	case <-time.After(eventually):
		t.Fatal("Recv returned nothing")
		return nil
	}
}

func recvErr(t *testing.T, ch link.DurableChannel) error {
	t.Helper()

	out := make(chan error, 1)

	go func() {
		_, err := ch.Recv()
		out <- err
	}()

	select {
	case err := <-out:
		require.Error(t, err)
		return err
	case <-time.After(eventually):
		t.Fatal("Recv returned nothing")
		return nil
	}
}

func openTestChannel(t *testing.T) (*registration, *fakeSession, link.DurableChannel) {
	t.Helper()

	session := &fakeSession{workerId: "w"}
	reg := &registration{session: session}

	ch, err := reg.OpenDurable(context.Background(), "task-1", 2)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = reg.Close()
		session.stream.end()
	})

	return reg, session, ch
}

func ref(task string, inv int32, branch, node int64) *v1.DurableEventLogEntryRef {
	return &v1.DurableEventLogEntryRef{DurableTaskExternalId: task, InvocationCount: inv, BranchId: branch, NodeId: node}
}

func TestDurableChannelMemo(t *testing.T) {
	_, session, ch := openTestChannel(t)

	// Ids are stamped even when the endpoint left them empty.
	require.NoError(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{
		Memo: &v1.DurableTaskMemoRequest{Key: []byte("k")},
	}}))

	sent := session.stream.next(t).GetMemo()
	require.NotNil(t, sent)
	assert.Equal(t, "task-1", sent.DurableTaskExternalId)
	assert.Equal(t, int32(2), sent.InvocationCount)

	session.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
		MemoAck: &v1.DurableTaskEventMemoAckResponse{Ref: ref("task-1", 2, 0, 7), MemoAlreadyExisted: true, MemoResultPayload: []byte(`{"v":1}`)},
	}}

	ack := recvOne(t, ch).GetMemoAck()
	require.NotNil(t, ack)
	assert.True(t, ack.MemoAlreadyExisted)
	assert.Equal(t, int64(7), ack.Ref.NodeId)

	// complete_memo is fire-and-forget and does not count as in flight.
	require.NoError(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_CompleteMemo{
		CompleteMemo: &v1.DurableTaskCompleteMemoRequest{Ref: ref("", 0, 0, 7), Payload: []byte(`{"v":2}`)},
	}}))

	complete := session.stream.next(t).GetCompleteMemo()
	require.NotNil(t, complete)
	assert.Equal(t, "task-1", complete.Ref.DurableTaskExternalId)
	assert.Equal(t, int32(2), complete.Ref.InvocationCount)
}

func TestDurableChannelWaitForDeliversEntryCompleted(t *testing.T) {
	_, session, ch := openTestChannel(t)

	require.NoError(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_WaitFor{
		WaitFor: &v1.DurableTaskWaitForRequest{},
	}}))

	require.NotNil(t, session.stream.next(t).GetWaitFor())

	session.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_WaitForAck{
		WaitForAck: &v1.DurableTaskEventWaitForAckResponse{Ref: ref("task-1", 2, 1, 3)},
	}}

	require.NotNil(t, recvOne(t, ch).GetWaitForAck())

	// The callback is registered under (task, invocation, branch, node) like the SDK's
	// WaitForCallback, and the worker status advertises it.
	require.Eventually(t, func() bool { return session.listeners[0].PendingCallbackCount() == 1 }, eventually, 10*time.Millisecond)

	session.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EntryCompleted{
		EntryCompleted: &v1.DurableTaskEventLogEntryCompletedResponse{Ref: ref("task-1", 2, 1, 3), Payload: []byte(`{"slept":true}`)},
	}}

	completed := recvOne(t, ch).GetEntryCompleted()
	require.NotNil(t, completed)
	assert.Equal(t, `{"slept":true}`, string(completed.Payload))
}

func TestDurableChannelTriggerRunsDeliversChildCompletions(t *testing.T) {
	_, session, ch := openTestChannel(t)

	require.NoError(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_TriggerRuns{
		TriggerRuns: &v1.DurableTaskTriggerRunsRequest{TriggerOpts: []*v1.TriggerWorkflowRequest{{Name: "child"}}},
	}}))

	require.NotNil(t, session.stream.next(t).GetTriggerRuns())

	session.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_TriggerRunsAck{
		TriggerRunsAck: &v1.DurableTaskEventTriggerRunsAckResponse{
			DurableTaskExternalId: "task-1",
			InvocationCount:       2,
			RunEntries:            []*v1.DurableTaskRunAckEntry{{NodeId: 5, BranchId: 0, WorkflowRunExternalId: "run-a"}},
		},
	}}

	require.NotNil(t, recvOne(t, ch).GetTriggerRunsAck())
	require.Eventually(t, func() bool { return session.listeners[0].PendingCallbackCount() == 1 }, eventually, 10*time.Millisecond)

	session.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EntryCompleted{
		EntryCompleted: &v1.DurableTaskEventLogEntryCompletedResponse{Ref: ref("task-1", 2, 0, 5), Payload: []byte(`{"child":1}`)},
	}}

	completed := recvOne(t, ch).GetEntryCompleted()
	require.NotNil(t, completed)
	assert.Equal(t, int64(5), completed.Ref.NodeId)
}

func TestDurableChannelEviction(t *testing.T) {
	_, session, ch := openTestChannel(t)

	require.NoError(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_EvictInvocation{
		EvictInvocation: &v1.DurableTaskEvictInvocationRequest{},
	}}))

	evict := session.stream.next(t).GetEvictInvocation()
	require.NotNil(t, evict)
	assert.Equal(t, "task-1", evict.DurableTaskExternalId)
	assert.Equal(t, int32(2), evict.InvocationCount)

	session.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EvictionAck{
		EvictionAck: &v1.DurableTaskEvictionAckResponse{DurableTaskExternalId: "task-1", InvocationCount: 2},
	}}

	ack := recvOne(t, ch).GetEvictionAck()
	require.NotNil(t, ack)
	assert.Equal(t, int32(2), ack.InvocationCount)
}

func TestDurableChannelServerEvict(t *testing.T) {
	_, session, ch := openTestChannel(t)

	// A memo is in flight when the engine supersedes the invocation: the listener fails the
	// ack during cleanup, which the eviction explains, so Recv yields server_evict, not an
	// error.
	require.NoError(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{
		Memo: &v1.DurableTaskMemoRequest{Key: []byte("k")},
	}}))
	session.stream.next(t)

	session.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_ServerEvict{
		ServerEvict: &v1.DurableTaskServerEvictNotice{DurableTaskExternalId: "task-1", InvocationCount: 2, Reason: "superseded"},
	}}

	notice := recvOne(t, ch).GetServerEvict()
	require.NotNil(t, notice)
	assert.Equal(t, "superseded", notice.Reason)
}

func TestDurableChannelNonDeterminismBecomesErrorResponse(t *testing.T) {
	_, session, ch := openTestChannel(t)

	require.NoError(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{
		Memo: &v1.DurableTaskMemoRequest{Key: []byte("k")},
	}}))
	session.stream.next(t)

	session.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_Error{
		Error: &v1.DurableTaskErrorResponse{
			Ref:          ref("task-1", 2, 0, 1),
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
	_, session, ch := openTestChannel(t)

	memo := func() *v1.DurableTaskRequest {
		return &v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{Memo: &v1.DurableTaskMemoRequest{Key: []byte("k")}}}
	}

	require.NoError(t, ch.Send(memo()))
	assert.ErrorIs(t, ch.Send(memo()), link.ErrRequestInFlight)
	assert.ErrorIs(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_EvictInvocation{
		EvictInvocation: &v1.DurableTaskEvictInvocationRequest{},
	}}), link.ErrRequestInFlight)

	session.stream.next(t)
	session.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
		MemoAck: &v1.DurableTaskEventMemoAckResponse{Ref: ref("task-1", 2, 0, 1)},
	}}
	recvOne(t, ch)

	// The slot is free again once the ack arrived.
	require.NoError(t, ch.Send(memo()))
}

func TestDurableChannelRejectsLinkInternalRequests(t *testing.T) {
	_, _, ch := openTestChannel(t)

	assert.Error(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_RegisterWorker{
		RegisterWorker: &v1.DurableTaskRequestRegisterWorker{WorkerId: "x"},
	}}))
	assert.Error(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_WorkerStatus{
		WorkerStatus: &v1.DurableTaskWorkerStatusRequest{WorkerId: "x"},
	}}))
}

func TestDurableChannelLinkFailureIsAnError(t *testing.T) {
	_, session, ch := openTestChannel(t)

	require.NoError(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{
		Memo: &v1.DurableTaskMemoRequest{Key: []byte("k")},
	}}))
	session.stream.next(t)

	// Stopping the listener fails the pending ack with no eviction to explain it.
	session.listeners[0].Stop()

	err := recvErr(t, ch)
	assert.Contains(t, err.Error(), "listener stopped")
}

func TestDurableChannelCloseCleansUp(t *testing.T) {
	reg, session, ch := openTestChannel(t)

	require.NoError(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_WaitFor{
		WaitFor: &v1.DurableTaskWaitForRequest{},
	}}))
	session.stream.next(t)
	session.stream.recv <- &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_WaitForAck{
		WaitForAck: &v1.DurableTaskEventWaitForAckResponse{Ref: ref("task-1", 2, 0, 3)},
	}}
	recvOne(t, ch)

	listener := session.listeners[0]
	require.Eventually(t, func() bool { return listener.PendingCallbackCount() == 1 }, eventually, 10*time.Millisecond)

	require.NoError(t, ch.Close())
	require.NoError(t, ch.Close(), "Close is idempotent")

	assert.Equal(t, 0, listener.PendingCallbackCount())
	assert.Equal(t, 0, listener.PendingEventAckCount())
	assert.ErrorIs(t, recvErr(t, ch), link.ErrChannelClosed)
	assert.ErrorIs(t, ch.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_Memo{
		Memo: &v1.DurableTaskMemoRequest{Key: []byte("k")},
	}}), link.ErrChannelClosed)

	// The same invocation can be reopened after Close; a duplicate open is refused.
	again, err := reg.OpenDurable(context.Background(), "task-1", 2)
	require.NoError(t, err)

	_, err = reg.OpenDurable(context.Background(), "task-1", 2)
	assert.Error(t, err)

	require.NoError(t, again.Close())

	// Closing the registration closes open channels and stops the listener.
	third, err := reg.OpenDurable(context.Background(), "task-2", 0)
	require.NoError(t, err)
	require.NoError(t, reg.Close())
	assert.ErrorIs(t, recvErr(t, third), link.ErrChannelClosed)
	assert.True(t, session.closed)
	assert.False(t, listener.IsRunning())

	_, err = reg.OpenDurable(context.Background(), "task-3", 0)
	assert.Error(t, err)
}
