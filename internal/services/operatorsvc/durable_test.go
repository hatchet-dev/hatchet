package operatorsvc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc/operatorsvctest"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func entryRef(branchId, nodeId int64) *v1contracts.DurableEventLogEntryRef {
	return &v1contracts.DurableEventLogEntryRef{BranchId: branchId, NodeId: nodeId}
}

func entryCompleted(branchId, nodeId int64) *v1contracts.DurableTaskResponse {
	return &v1contracts.DurableTaskResponse{Message: &v1contracts.DurableTaskResponse_EntryCompleted{
		EntryCompleted: &v1contracts.DurableTaskEventLogEntryCompletedResponse{Ref: entryRef(branchId, nodeId)},
	}}
}

func waitForAck(branchId, nodeId int64) *v1contracts.DurableTaskResponse {
	return &v1contracts.DurableTaskResponse{Message: &v1contracts.DurableTaskResponse_WaitForAck{
		WaitForAck: &v1contracts.DurableTaskEventWaitForAckResponse{Ref: entryRef(branchId, nodeId)},
	}}
}

func registerWorkerAck(workerId string) *v1contracts.DurableTaskResponse {
	return &v1contracts.DurableTaskResponse{Message: &v1contracts.DurableTaskResponse_RegisterWorker{
		RegisterWorker: &v1contracts.DurableTaskResponseRegisterWorker{WorkerId: workerId},
	}}
}

func memoAck() *v1contracts.DurableTaskResponse {
	return &v1contracts.DurableTaskResponse{Message: &v1contracts.DurableTaskResponse_MemoAck{
		MemoAck: &v1contracts.DurableTaskEventMemoAckResponse{Ref: entryRef(0, 1)},
	}}
}

func memoRequest() *v1contracts.DurableTaskRequest {
	return &v1contracts.DurableTaskRequest{Message: &v1contracts.DurableTaskRequest_Memo{
		Memo: &v1contracts.DurableTaskMemoRequest{Key: []byte("k")},
	}}
}

func waitForRequest() *v1contracts.DurableTaskRequest {
	return &v1contracts.DurableTaskRequest{Message: &v1contracts.DurableTaskRequest_WaitFor{
		WaitFor: &v1contracts.DurableTaskWaitForRequest{},
	}}
}

// durableSession opens a session with a handler delivery and returns it with the task id an
// invocation would be opened for.
func durableSession(t *testing.T, svc *testService, tenant *sqlcv1.Tenant) (*operatorsvc.Session, uuid.UUID) {
	t.Helper()

	op, worker := registeredOperator(t, svc, tenant)

	session, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	t.Cleanup(func() { _ = session.Close(t.Context()) })

	return session, worker.ID
}

// openInvocation drives the handshake in the background: it waits for the register request and
// answers it with the ack, after feeding the given responses first.
func openInvocation(t *testing.T, svc *testService, session *operatorsvc.Session, workerId uuid.UUID, before ...*v1contracts.DurableTaskResponse) (operatorsvc.DurableChannel, *operatorsvctest.DurableInvocation) {
	t.Helper()

	taskId := uuid.New()
	done := make(chan struct{})

	go func() {
		defer close(done)

		eventually(t, func() bool { return len(svc.dispatcher.Durables()) > 0 }, "the invocation was never registered")

		inv := svc.dispatcher.Durables()[0]

		select {
		case req := <-inv.Requests:
			assert.Equal(t, workerId.String(), req.GetRegisterWorker().GetWorkerId(), "the handshake names the session's worker")
		case <-time.After(5 * time.Second):
			t.Error("no register worker request")
			return
		}

		for _, resp := range before {
			inv.Responses <- resp
		}

		inv.Responses <- registerWorkerAck(workerId.String())
	}()

	ch, err := session.OpenDurable(t.Context(), taskId, 3)
	require.NoError(t, err)

	<-done

	invocations := svc.dispatcher.Durables()
	require.Len(t, invocations, 1)
	assert.Equal(t, taskId, invocations[0].ExternalId)

	t.Cleanup(func() { _ = ch.Close() })

	return ch, invocations[0]
}

// The handshake is done by the session: Recv only ever returns invocation traffic, and requests
// carry the invocation's identity.
func TestOpenDurableHandshakeAndStamping(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, workerId := durableSession(t, svc, tenant)

	ch, inv := openInvocation(t, svc, session, workerId)

	require.NoError(t, ch.Send(t.Context(), memoRequest()))

	req := <-inv.Requests
	assert.Equal(t, inv.ExternalId.String(), req.GetMemo().GetDurableTaskExternalId())
	assert.Equal(t, int32(3), req.GetMemo().GetInvocationCount())

	// one ack-bearing request at a time: the engine keys its pending state by (task, invocation)
	assert.ErrorIs(t, ch.Send(t.Context(), memoRequest()), operatorsvc.ErrRequestInFlight)

	inv.Responses <- memoAck()

	resp, err := ch.Recv(t.Context())
	require.NoError(t, err)
	assert.NotNil(t, resp.GetMemoAck())

	require.NoError(t, ch.Send(t.Context(), memoRequest()), "the slot is released when the ack is read")
}

// register_worker belongs to the session, not to the operator driving the invocation; a
// worker_status the operator sends to report what it is blocked on passes through, naming the
// session's worker whatever the operator put in it.
func TestOpenDurableSessionOwnedRequests(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, workerId := durableSession(t, svc, tenant)

	ch, inv := openInvocation(t, svc, session, workerId)

	err := ch.Send(t.Context(), &v1contracts.DurableTaskRequest{Message: &v1contracts.DurableTaskRequest_RegisterWorker{
		RegisterWorker: &v1contracts.DurableTaskRequestRegisterWorker{WorkerId: workerId.String()},
	}})
	require.Error(t, err)

	require.NoError(t, ch.Send(t.Context(), &v1contracts.DurableTaskRequest{Message: &v1contracts.DurableTaskRequest_WorkerStatus{
		WorkerStatus: &v1contracts.DurableTaskWorkerStatusRequest{WorkerId: "someone-else"},
	}}))

	req := <-inv.Requests
	assert.Equal(t, workerId.String(), req.GetWorkerStatus().GetWorkerId())
}

// The engine delivers responses on one goroutine that blocks on each delivery before it reads
// the next request; an operator that sends while a response is undelivered must not deadlock
// against it.
func TestOpenDurableSendWhileResponsesPending(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, workerId := durableSession(t, svc, tenant)

	ch, inv := openInvocation(t, svc, session, workerId)

	// the engine has responses queued that nobody has read yet
	for i := 0; i < 8; i++ {
		inv.Responses <- memoAck()
	}

	done := make(chan error, 1)

	go func() { done <- ch.Send(t.Context(), memoRequest()) }()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Send blocked behind undelivered responses")
	}

	<-inv.Requests

	for i := 0; i < 8; i++ {
		resp, err := ch.Recv(t.Context())
		require.NoError(t, err)
		require.NotNil(t, resp.GetMemoAck())
	}
}

// Recv and Send honour the caller's context.
func TestOpenDurableContextEnds(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, workerId := durableSession(t, svc, tenant)

	ch, _ := openInvocation(t, svc, session, workerId)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	_, err := ch.Recv(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// the in-flight slot is given back when a send is abandoned
	sendCtx, cancelSend := context.WithCancel(t.Context())
	cancelSend()

	assert.ErrorIs(t, ch.Send(sendCtx, memoRequest()), context.Canceled)
	require.NoError(t, ch.Send(t.Context(), memoRequest()))
}

// When the engine ends the invocation on its own, Recv reports it as the session ending, not as
// the channel being closed.
func TestOpenDurableEngineEndsSession(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, workerId := durableSession(t, svc, tenant)

	ch, inv := openInvocation(t, svc, session, workerId)

	inv.Responses <- memoAck()
	inv.End()

	resp, err := ch.Recv(t.Context())
	require.NoError(t, err)
	require.NotNil(t, resp.GetMemoAck(), "what the engine sent before ending is still delivered")

	_, err = ch.Recv(t.Context())
	assert.ErrorIs(t, err, operatorsvc.ErrSessionEnded)
}

// The engine delivers a completion for an already satisfied entry as soon as the invocation
// registers, which on a replay is before the operator sent the wait_for that names it. The
// operator expects the ack first, so the entry is held until its ack has been returned.
func TestOpenDurableHoldsEntriesUntilTheirAck(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, workerId := durableSession(t, svc, tenant)

	ch, inv := openInvocation(t, svc, session, workerId)

	inv.Responses <- entryCompleted(0, 7)

	require.NoError(t, ch.Send(t.Context(), waitForRequest()))
	<-inv.Requests

	received := make(chan *v1contracts.DurableTaskResponse, 2)
	go func() {
		for i := 0; i < 2; i++ {
			resp, err := ch.Recv(t.Context())

			if err != nil {
				return
			}

			received <- resp
		}
	}()

	select {
	case resp := <-received:
		t.Fatalf("an entry was delivered before its ack: %v", resp)
	case <-time.After(100 * time.Millisecond):
	}

	inv.Responses <- waitForAck(0, 7)

	first := <-received
	assert.NotNil(t, first.GetWaitForAck(), "the ack comes first")

	second := <-received
	require.NotNil(t, second.GetEntryCompleted())
	assert.Equal(t, int64(7), second.GetEntryCompleted().GetRef().GetNodeId(), "the held entry follows its ack")
}

// An entry whose ack already went out is delivered as it arrives.
func TestOpenDurableDeliversEntriesAfterTheirAck(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, workerId := durableSession(t, svc, tenant)

	ch, inv := openInvocation(t, svc, session, workerId)

	require.NoError(t, ch.Send(t.Context(), waitForRequest()))
	<-inv.Requests

	inv.Responses <- waitForAck(1, 4)
	inv.Responses <- entryCompleted(1, 4)

	ack, err := ch.Recv(t.Context())
	require.NoError(t, err)
	require.NotNil(t, ack.GetWaitForAck())

	entry, err := ch.Recv(t.Context())
	require.NoError(t, err)
	require.NotNil(t, entry.GetEntryCompleted())
	assert.Equal(t, int64(4), entry.GetEntryCompleted().GetRef().GetNodeId())
}

// Traffic that arrives during the handshake is held for the channel under the same ordering.
func TestOpenDurableHoldsPreHandshakeResponses(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, workerId := durableSession(t, svc, tenant)

	ch, inv := openInvocation(t, svc, session, workerId, entryCompleted(0, 2))

	require.NoError(t, ch.Send(t.Context(), waitForRequest()))
	<-inv.Requests

	inv.Responses <- waitForAck(0, 2)

	ack, err := ch.Recv(t.Context())
	require.NoError(t, err)
	require.NotNil(t, ack.GetWaitForAck())

	entry, err := ch.Recv(t.Context())
	require.NoError(t, err)
	require.NotNil(t, entry.GetEntryCompleted(), "the entry held through the handshake is delivered behind its ack")
	assert.Equal(t, int64(2), entry.GetEntryCompleted().GetRef().GetNodeId())
}

// The hold is bounded: an engine that never acks the handshake cannot make the session buffer
// without end.
func TestOpenDurableHoldLimit(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, _ := durableSession(t, svc, tenant)

	go func() {
		eventually(t, func() bool { return len(svc.dispatcher.Durables()) > 0 }, "the invocation was never registered")

		inv := svc.dispatcher.Durables()[0]
		<-inv.Requests

		for i := 0; i <= operatorsvc.HandshakeHoldLimit; i++ {
			select {
			case inv.Responses <- entryCompleted(0, int64(i)):
			case <-time.After(5 * time.Second):
				return
			}
		}
	}()

	_, err := session.OpenDurable(t.Context(), uuid.New(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "before the register worker ack")
}

// An engine that refuses the handshake fails the open rather than leaving a half-open channel.
func TestOpenDurableRejectedHandshake(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, _ := durableSession(t, svc, tenant)

	go func() {
		eventually(t, func() bool { return len(svc.dispatcher.Durables()) > 0 }, "the invocation was never registered")

		inv := svc.dispatcher.Durables()[0]
		<-inv.Requests

		inv.Responses <- &v1contracts.DurableTaskResponse{Message: &v1contracts.DurableTaskResponse_Error{
			Error: &v1contracts.DurableTaskErrorResponse{ErrorMessage: "worker is not registered"},
		}}
	}()

	_, err := session.OpenDurable(t.Context(), uuid.New(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worker is not registered")
}

func TestOpenDurableRegisterFailure(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, _ := durableSession(t, svc, tenant)

	svc.dispatcher.FailRegisterDurableTask(errors.New("no such task"))

	_, err := session.OpenDurable(t.Context(), uuid.New(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such task")
}

// Close ends the channel for both directions.
func TestOpenDurableCloseUnblocks(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, workerId := durableSession(t, svc, tenant)

	ch, _ := openInvocation(t, svc, session, workerId)

	require.NoError(t, ch.Close())

	_, err := ch.Recv(t.Context())
	assert.ErrorIs(t, err, operatorsvc.ErrChannelClosed)
	assert.ErrorIs(t, ch.Send(t.Context(), memoRequest()), operatorsvc.ErrChannelClosed)
}

// An entry the operator learned of outside the channel (a DAG child created by the direct
// trigger) has no ack: ExpectEntry stands in for it, releasing a completion that already
// arrived and letting a later one through as it comes.
func TestOpenDurableExpectEntryStandsInForTheAck(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, workerId := durableSession(t, svc, tenant)

	ch, inv := openInvocation(t, svc, session, workerId)

	inv.Responses <- entryCompleted(1, 1)

	recvCtx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	_, err := ch.Recv(recvCtx)
	cancel()
	require.ErrorIs(t, err, context.DeadlineExceeded, "an entry nothing acknowledged is held")

	require.NoError(t, ch.ExpectEntry(1, 1))

	held, err := ch.Recv(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int64(1), held.GetEntryCompleted().GetRef().GetNodeId(), "the held completion is released")

	require.NoError(t, ch.ExpectEntry(1, 2))
	inv.Responses <- entryCompleted(1, 2)

	later, err := ch.Recv(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int64(2), later.GetEntryCompleted().GetRef().GetNodeId(), "a completion after the expectation is delivered as it comes")
}

// A completion is delivered once: the engine resends satisfied entries the operator reports as
// awaited, and a repeat must neither be delivered again nor be retained.
func TestOpenDurableDeliversACompletionOnce(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, workerId := durableSession(t, svc, tenant)

	ch, inv := openInvocation(t, svc, session, workerId)

	require.NoError(t, ch.ExpectEntry(0, 5))
	inv.Responses <- entryCompleted(0, 5)
	inv.Responses <- entryCompleted(0, 5)

	first, err := ch.Recv(t.Context())
	require.NoError(t, err)
	require.NotNil(t, first.GetEntryCompleted())

	recvCtx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	_, err = ch.Recv(recvCtx)
	cancel()
	require.ErrorIs(t, err, context.DeadlineExceeded, "the repeat is dropped")

	require.NoError(t, ch.ExpectEntry(0, 5), "expecting a delivered entry again changes nothing")
	assert.Zero(t, operatorsvc.RetainedResponses(ch), "nothing is retained for a delivered entry")
}

// What the pump retains for Recv, deliverable or held, is bounded for the life of the
// invocation: past the limit the channel fails rather than buffer without end.
func TestOpenDurableRetainedResponsesAreBounded(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	session, workerId := durableSession(t, svc, tenant)

	ch, inv := openInvocation(t, svc, session, workerId)

	fed := make(chan struct{})

	go func() {
		defer close(fed)

		for i := 0; i <= operatorsvc.RetainedResponseLimit; i++ {
			select {
			case inv.Responses <- entryCompleted(0, int64(i)):
			case <-t.Context().Done():
				return
			}
		}
	}()

	select {
	case <-fed:
	case <-time.After(5 * time.Second):
		t.Fatal("the engine could not hand its responses to the pump")
	}

	recvCtx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	_, err := ch.Recv(recvCtx)
	require.ErrorIs(t, err, operatorsvc.ErrChannelClosed, "a channel past its retention limit fails")

	require.ErrorIs(t, ch.Send(t.Context(), waitForRequest()), operatorsvc.ErrChannelClosed)
	assert.Zero(t, operatorsvc.RetainedResponses(ch), "a failed channel retains nothing")
}
