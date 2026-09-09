package grpcoperator

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// fakeListenStream feeds client messages to the handler. Closing the queue reads as the client
// closing its send side (io.EOF).
type fakeListenStream struct {
	grpc.ServerStream

	ctx  context.Context
	recv chan *v1contracts.OperatorListenRequest
}

func newFakeListenStream(ctx context.Context, msgs ...*v1contracts.OperatorListenRequest) *fakeListenStream {
	recv := make(chan *v1contracts.OperatorListenRequest, 64)

	for _, m := range msgs {
		recv <- m
	}

	return &fakeListenStream{ctx: ctx, recv: recv}
}

func (f *fakeListenStream) Context() context.Context { return f.ctx }

func (f *fakeListenStream) Recv() (*v1contracts.OperatorListenRequest, error) {
	select {
	case msg, ok := <-f.recv:
		if !ok {
			return nil, io.EOF
		}

		return msg, nil
	case <-f.ctx.Done():
		return nil, status.Error(codes.Canceled, "stream context done")
	}
}

func (f *fakeListenStream) Send(*v1contracts.OperatorListenResponse) error { return nil }

func (f *fakeListenStream) push(msg *v1contracts.OperatorListenRequest) { f.recv <- msg }

func heartbeatMsg() *v1contracts.OperatorListenRequest {
	return &v1contracts.OperatorListenRequest{Message: &v1contracts.OperatorListenRequest_Heartbeat{
		Heartbeat: &v1contracts.OperatorHeartbeat{HeartbeatAt: timestamppb.Now()},
	}}
}

func startMsg(workerId string) *v1contracts.OperatorListenRequest {
	return &v1contracts.OperatorListenRequest{Message: &v1contracts.OperatorListenRequest_Start{
		Start: &v1contracts.OperatorListenStart{WorkerId: workerId},
	}}
}

func deltaMsg(add, remove []string) *v1contracts.OperatorListenRequest {
	return &v1contracts.OperatorListenRequest{Message: &v1contracts.OperatorListenRequest_Actions{
		Actions: &v1contracts.OperatorActionsDelta{Add: add, Remove: remove},
	}}
}

// runListen runs the handler in the background and returns a channel with its result.
func runListen(svc *testService, stream *fakeListenStream) <-chan error {
	done := make(chan error, 1)

	go func() { done <- svc.Listen(stream) }()

	return done
}

func waitListen(t *testing.T, done <-chan error) error {
	t.Helper()

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("Listen did not return")
		return nil
	}
}

func TestListenRequiresOperatorMetadata(t *testing.T) {
	ctx, cancel := context.WithCancel(tenantContext(&sqlcv1.Tenant{ID: uuid.New()}))
	defer cancel()

	svc := newTestService(t, nil)

	err := svc.Listen(newFakeListenStream(ctx, startMsg(uuid.NewString())))

	assert.Equal(t, codes.InvalidArgument, status.Code(err), err)
	assert.Zero(t, svc.dispatcher.SessionCount())
}

func TestListenRejectsNonStartFirstMessage(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, _ := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, first := range []*v1contracts.OperatorListenRequest{heartbeatMsg(), deltaMsg([]string{"svc:a"}, nil)} {
		err := svc.Listen(newFakeListenStream(ctx, first))

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err), err.Error())
	}

	assert.Zero(t, svc.dispatcher.SessionCount(), "nothing is registered before a start message")
	assert.Empty(t, svc.workers.Activations())
}

func TestListenRejectsForeignWorker(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, _ := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	otherOp := uuid.New()
	other := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &otherOp})

	err := svc.Listen(newFakeListenStream(ctx, startMsg(other.ID.String())))
	assert.Equal(t, codes.PermissionDenied, status.Code(err), err)

	err = svc.Listen(newFakeListenStream(ctx, startMsg("nope")))
	assert.Equal(t, codes.InvalidArgument, status.Code(err), err)

	assert.Zero(t, svc.dispatcher.SessionCount())
}

func TestListenClientCloseBeforeStart(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, _ := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream := newFakeListenStream(ctx)
	close(stream.recv)

	assert.NoError(t, svc.Listen(stream))
}

func TestListenActivatesAndDeactivatesWithSessionId(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, worker := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream := newFakeListenStream(ctx, startMsg(worker.ID.String()))
	done := runListen(svc, stream)

	eventually(t, func() bool { return svc.dispatcher.SessionCount() == 1 }, "session was not registered")
	assert.Equal(t, []uuid.UUID{worker.ID}, svc.workers.Activations())
	assert.Equal(t, 1, svc.dispatcher.NotifyCount(), "a new session notifies the scheduler once")
	assert.Equal(t, svc.dispatcherId, svc.workers.DispatcherFor(worker.ID), "the worker is pinned to this dispatcher")

	// heartbeats are written at most once per second
	stream.push(heartbeatMsg())
	stream.push(heartbeatMsg())
	eventually(t, func() bool { return svc.workers.Heartbeats() == 1 }, "heartbeat was not written")

	// a second start is a protocol error
	stream.push(startMsg(worker.ID.String()))

	err := waitListen(t, done)
	assert.Equal(t, codes.InvalidArgument, status.Code(err), err)
	assert.Equal(t, 1, svc.dispatcher.ReleasedCount(), "the session is released on exit")

	sessions := svc.workers.SessionLog()
	require.Len(t, sessions, 2, "the worker is activated on start and deactivated on exit")
	assert.Equal(t, sessions[0], sessions[1], "deactivation is fenced on the activation session id")
	assert.Equal(t, []uuid.UUID{sessions[0]}, svc.dispatcher.SessionIdLog(), "the dispatcher session is keyed on the listener session id")
	assert.False(t, svc.workers.IsActive(worker.ID), "the worker is inactive once its only session ends")
}

// A newer Listen stream on the same worker takes over the listener session. When the older
// stream exits afterwards, its deactivation is superseded and must leave the worker active
// for the session that now owns it.
func TestListenSupersededSessionLeavesWorkerActive(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, worker := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	first := newFakeListenStream(ctx, startMsg(worker.ID.String()))
	firstDone := runListen(svc, first)

	eventually(t, func() bool { return svc.dispatcher.SessionCount() == 1 }, "first session was not registered")

	second := newFakeListenStream(ctx, startMsg(worker.ID.String()))
	secondDone := runListen(svc, second)

	eventually(t, func() bool { return svc.dispatcher.SessionCount() == 2 }, "second session was not registered")

	close(first.recv)
	assert.NoError(t, waitListen(t, firstDone), "a superseded deactivation is not an error")
	assert.True(t, svc.workers.IsActive(worker.ID), "a superseded session must not deactivate the worker")

	close(second.recv)
	assert.NoError(t, waitListen(t, secondDone))
	assert.False(t, svc.workers.IsActive(worker.ID), "the live session's deactivation marks the worker inactive")

	sessions := svc.workers.SessionLog()
	require.Len(t, sessions, 4, "two activations and two deactivations")
	assert.Equal(t, sessions[:2], svc.dispatcher.SessionIdLog(), "each stream registers under its own listener session id")
	assert.NotEqual(t, sessions[0], sessions[1], "each stream gets its own session id")
	assert.Equal(t, sessions[0], sessions[2], "the first stream deactivates with its own session id")
	assert.Equal(t, sessions[1], sessions[3], "the second stream deactivates with its own session id")
}

func TestListenAppliesDeltasWithThrottledNotify(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil, operatorsvc.WithNotifyInterval(100*time.Millisecond))
	ctx, _, worker := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream := newFakeListenStream(ctx, startMsg(worker.ID.String()))
	done := runListen(svc, stream)

	eventually(t, func() bool { return svc.dispatcher.NotifyCount() == 1 }, "start did not notify")

	// deltas right after the start notify are inside the throttle window, so they are deferred
	// and folded into one notification
	stream.push(deltaMsg([]string{"svc:a", "svc:b"}, nil))
	stream.push(deltaMsg([]string{"svc:c"}, []string{"svc:b"}))

	eventually(t, func() bool { return len(svc.workers.ActionSet(worker.ID)) == 2 }, "deltas did not reach the store")
	assert.ElementsMatch(t, []string{"svc:a", "svc:c"}, svc.workers.ActionSet(worker.ID))
	assert.Equal(t, 1, svc.dispatcher.NotifyCount(), "deltas inside the window are not notified immediately")

	eventually(t, func() bool { return svc.dispatcher.NotifyCount() == 2 }, "deferred notify did not fire")

	// no further notify without further changes
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, 2, svc.dispatcher.NotifyCount())

	// a delta that changes nothing does not notify at all
	stream.push(deltaMsg([]string{"svc:a"}, []string{"svc:never"}))
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, 2, svc.dispatcher.NotifyCount())

	// a delta outside the window notifies immediately
	stream.push(deltaMsg(nil, []string{"svc:a"}))
	eventually(t, func() bool { return svc.dispatcher.NotifyCount() == 3 }, "delta outside the window did not notify")
	eventually(t, func() bool { return len(svc.workers.ActionSet(worker.ID)) == 1 }, "removal did not reach the store")
	assert.ElementsMatch(t, []string{"svc:c"}, svc.workers.ActionSet(worker.ID))

	close(stream.recv)
	assert.NoError(t, waitListen(t, done))
}

func TestListenRejectsBadDeltas(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}

	tooMany := make([]string, operatorsvc.MaxActionsPerDelta+1)

	for i := range tooMany {
		tooMany[i] = "svc:a"
	}

	cases := []struct {
		delta *v1contracts.OperatorListenRequest
		name  string
	}{
		{name: "over the cap", delta: deltaMsg(tooMany[:600], tooMany[600:])},
		{name: "invalid add", delta: deltaMsg([]string{"not an action"}, nil)},
		{name: "invalid remove", delta: deltaMsg(nil, []string{"not an action"})},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t, nil)
			ctx, _, worker := registeredOperator(t, svc, tenant)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			stream := newFakeListenStream(ctx, startMsg(worker.ID.String()), tc.delta)

			err := waitListen(t, runListen(svc, stream))
			assert.Equal(t, codes.InvalidArgument, status.Code(err), err)
			assert.Empty(t, svc.workers.ActionSet(worker.ID), "a rejected delta must not touch the store")
			assert.Len(t, svc.workers.SessionLog(), 2, "the worker is deactivated on error exit")
		})
	}

	t.Run("exactly the cap is accepted", func(t *testing.T) {
		svc := newTestService(t, nil)
		ctx, _, worker := registeredOperator(t, svc, tenant)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		stream := newFakeListenStream(ctx, startMsg(worker.ID.String()), deltaMsg(tooMany[:operatorsvc.MaxActionsPerDelta], nil))
		done := runListen(svc, stream)

		eventually(t, func() bool { return len(svc.workers.ActionSet(worker.ID)) == 1 }, "delta at the cap was not applied")

		close(stream.recv)
		assert.NoError(t, waitListen(t, done))
	})
}

func TestListenExitsOnDispatcherFin(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	ctx, _, worker := registeredOperator(t, svc, tenant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream := newFakeListenStream(ctx, startMsg(worker.ID.String()))
	done := runListen(svc, stream)

	eventually(t, func() bool { return svc.dispatcher.SessionCount() == 1 }, "session was not registered")

	svc.dispatcher.Fin() <- true

	assert.NoError(t, waitListen(t, done))
	assert.Len(t, svc.workers.SessionLog(), 2)
}
