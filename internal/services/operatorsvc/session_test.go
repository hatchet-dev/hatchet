package operatorsvc_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// A stream-backed session is the gRPC host's: it pins the worker here, activates it under one
// id that is both the dispatcher's session key and the row's listener fence, notifies the
// scheduler once, and takes a slot of the operator's stream cap.
func TestOpenSessionStreamDelivery(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	op, worker := registeredOperator(t, svc, tenant)

	session, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Stream: nopStream{}})
	require.NoError(t, err)

	assert.Equal(t, []uuid.UUID{worker.ID}, svc.workers.Activations())
	assert.Equal(t, []uuid.UUID{session.SessionId()}, svc.workers.SessionLog())
	assert.Equal(t, []uuid.UUID{session.SessionId()}, svc.dispatcher.SessionIdLog(), "the dispatcher session is keyed on the listener session id")
	assert.Equal(t, svc.dispatcherId, svc.workers.DispatcherFor(worker.ID), "the worker is pinned to this dispatcher")
	assert.Equal(t, 1, svc.dispatcher.NotifyCount(), "a new session notifies the scheduler once")
	assert.Equal(t, 1, svc.ListenStreamCount(op.ID))
	assert.NotNil(t, session.Fin())

	require.NoError(t, session.Close(t.Context(), operatorsvc.WithoutPause()))

	assert.False(t, svc.workers.IsActive(worker.ID))
	assert.Equal(t, 1, svc.dispatcher.ReleasedCount())
	assert.Zero(t, svc.ListenStreamCount(op.ID), "the stream slot is returned")
	assert.Equal(t, []uuid.UUID{session.SessionId(), session.SessionId()}, svc.workers.SessionLog(), "deactivation is fenced on the activation session id")
}

// A handler-backed session is the in-process host's: same fence, same notification, no stream,
// so no stream cap and nothing to send protocol messages on.
func TestOpenSessionHandlerDelivery(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	op, worker := registeredOperator(t, svc, tenant)

	handler := nopHandler{}

	session, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Handler: handler})
	require.NoError(t, err)

	assert.Equal(t, []uuid.UUID{session.SessionId()}, svc.workers.SessionLog())
	assert.Equal(t, []uuid.UUID{session.SessionId()}, svc.dispatcher.SessionIdLog())
	require.Len(t, svc.dispatcher.Handlers(), 1)
	assert.Equal(t, handler, svc.dispatcher.Handlers()[0], "the dispatcher delivers to the operator directly")
	assert.Zero(t, svc.ListenStreamCount(op.ID), "an in-process session holds no stream")
	assert.Nil(t, session.Fin(), "there is no stream to hang up")
	assert.ErrorIs(t, session.Send(t.Context(), nil), operatorsvc.ErrNoStream)

	require.NoError(t, session.Close(t.Context()))
	assert.False(t, svc.workers.IsActive(worker.ID))
}

func TestOpenSessionRequiresOneDelivery(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	op, worker := registeredOperator(t, svc, tenant)

	_, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{})
	require.Error(t, err)

	_, err = svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Stream: nopStream{}, Handler: nopHandler{}})
	require.Error(t, err)

	assert.Empty(t, svc.workers.SessionLog(), "a refused session never touches the worker")
}

// The stream cap is per operator per replica, and only stream-backed sessions count against it.
func TestOpenSessionStreamCap(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil, operatorsvc.WithMaxListenStreamsPerOperator(1))
	op, worker := registeredOperator(t, svc, tenant)

	second := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &op.ID})

	first, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Stream: nopStream{}})
	require.NoError(t, err)

	_, err = svc.OpenSession(t.Context(), tenant, op, second.ID, operatorsvc.OpenOpts{Stream: nopStream{}})
	assert.Equal(t, codes.ResourceExhausted, status.Code(err), err)
	assert.Len(t, svc.workers.SessionLog(), 1, "a refused stream never activates its worker")

	inProcess, err := svc.OpenSession(t.Context(), tenant, op, second.ID, operatorsvc.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err, "an in-process session is not bounded by the stream cap")
	require.NoError(t, inProcess.Close(t.Context()))

	require.NoError(t, first.Close(t.Context(), operatorsvc.WithoutPause()))

	third, err := svc.OpenSession(t.Context(), tenant, op, second.ID, operatorsvc.OpenOpts{Stream: nopStream{}})
	require.NoError(t, err, "the slot is released when the session closes")
	require.NoError(t, third.Close(t.Context(), operatorsvc.WithoutPause()))
}

// Close is pause then drain: the scheduler stops assigning before the delivery goes away, so a
// host that drains its operator between Pause and Close sees no new work.
func TestSessionCloseIsPauseThenDrain(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	op, worker := registeredOperator(t, svc, tenant)

	session, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	require.NoError(t, session.Close(t.Context()))

	assert.True(t, svc.workers.IsPaused(worker.ID))
	assert.False(t, svc.workers.IsActive(worker.ID))
	assert.Equal(t, []string{"activate", "pause", "deactivate"}, svc.workers.Ops(), "the pause is committed before the worker is deactivated")
}

// A host that pauses for itself, then drains, does not pay for a second write on close.
func TestSessionCloseAfterPauseDoesNotPauseAgain(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	op, worker := registeredOperator(t, svc, tenant)

	session, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	require.NoError(t, session.Pause(t.Context(), true))
	require.NoError(t, session.Close(t.Context()))

	assert.Equal(t, []string{"activate", "pause", "deactivate"}, svc.workers.Ops())
}

// The gRPC host closes without a pause: its operator pauses itself through PauseWorker, and a
// stream that simply drops must leave the worker assignable for the next connection.
func TestSessionCloseWithoutPause(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	op, worker := registeredOperator(t, svc, tenant)

	session, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Stream: nopStream{}})
	require.NoError(t, err)

	require.NoError(t, session.Close(t.Context(), operatorsvc.WithoutPause()))

	assert.False(t, svc.workers.IsPaused(worker.ID))
	assert.Equal(t, []string{"activate", "deactivate"}, svc.workers.Ops())

	// closing again is a no-op: the session is already gone
	require.NoError(t, session.Close(t.Context(), operatorsvc.WithoutPause()))
	assert.Equal(t, []string{"activate", "deactivate"}, svc.workers.Ops())
	assert.Equal(t, 1, svc.dispatcher.ReleasedCount())
}

// A newer session on the same worker takes over the listener fence. When the older one closes
// afterwards, its deactivation is superseded and must leave the worker active.
func TestSessionSupersededCloseLeavesWorkerActive(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	op, worker := registeredOperator(t, svc, tenant)

	first, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Stream: nopStream{}})
	require.NoError(t, err)

	second, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Stream: nopStream{}})
	require.NoError(t, err)

	assert.NotEqual(t, first.SessionId(), second.SessionId())

	require.NoError(t, first.Close(t.Context(), operatorsvc.WithoutPause()), "a superseded deactivation is not an error")
	assert.True(t, svc.workers.IsActive(worker.ID), "a superseded session must not deactivate the worker")

	require.NoError(t, second.Close(t.Context(), operatorsvc.WithoutPause()))
	assert.False(t, svc.workers.IsActive(worker.ID))
}

func TestSessionHeartbeatIsThrottled(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	op, worker := registeredOperator(t, svc, tenant)

	session, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Stream: nopStream{}})
	require.NoError(t, err)

	now := time.Now().UTC()

	require.NoError(t, session.Heartbeat(t.Context(), now))
	require.NoError(t, session.Heartbeat(t.Context(), now.Add(100*time.Millisecond)))
	assert.Equal(t, 1, svc.workers.Heartbeats(), "heartbeats are written at most once per second")

	require.NoError(t, session.Heartbeat(t.Context(), now.Add(2*time.Second)))
	assert.Equal(t, 2, svc.workers.Heartbeats())

	require.NoError(t, session.Close(t.Context(), operatorsvc.WithoutPause()))
}

func TestSessionApplyDeltaRejectsBadDeltas(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}

	tooMany := make([]string, operatorsvc.MaxActionsPerDelta+1)

	for i := range tooMany {
		tooMany[i] = "svc:a"
	}

	cases := []struct {
		name   string
		add    []string
		remove []string
	}{
		{name: "over the cap", add: tooMany[:600], remove: tooMany[600:]},
		{name: "invalid add", add: []string{"not an action"}},
		{name: "invalid remove", remove: []string{"not an action"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t, nil)
			op, worker := registeredOperator(t, svc, tenant)

			session, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Stream: nopStream{}})
			require.NoError(t, err)

			_, err = session.ApplyDelta(t.Context(), tc.add, tc.remove)
			assert.Equal(t, codes.InvalidArgument, status.Code(err), err)
			assert.Empty(t, svc.workers.ActionSet(worker.ID), "a rejected delta must not touch the store")

			require.NoError(t, session.Close(t.Context(), operatorsvc.WithoutPause()))
		})
	}

	t.Run("exactly the cap is accepted", func(t *testing.T) {
		svc := newTestService(t, nil)
		op, worker := registeredOperator(t, svc, tenant)

		session, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Stream: nopStream{}})
		require.NoError(t, err)

		changed, err := session.ApplyDelta(t.Context(), tooMany[:operatorsvc.MaxActionsPerDelta], nil)
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Len(t, svc.workers.ActionSet(worker.ID), 1)

		require.NoError(t, session.Close(t.Context(), operatorsvc.WithoutPause()))
	})
}

// The action link budget spans every worker of the operator and applies to both deliveries: an
// in-process operator can exhaust the tenant's links the same way an out-of-process one can.
func TestSessionApplyDeltaBudget(t *testing.T) {
	for _, delivery := range []string{"stream", "handler"} {
		t.Run(delivery, func(t *testing.T) {
			tenant := &sqlcv1.Tenant{ID: uuid.New()}
			svc := newTestService(t, nil, operatorsvc.WithMaxActionsPerOperator(3))
			op, worker := registeredOperator(t, svc, tenant)

			// links held by another worker of the same operator count against the budget
			sibling := svc.workers.Add(&sqlcv1.Worker{ID: uuid.New(), TenantId: tenant.ID, OperatorId: &op.ID})
			_, err := svc.workers.AddWorkerActions(t.Context(), tenant.ID, sibling.ID, []string{"svc:held"})
			require.NoError(t, err)

			opts := operatorsvc.OpenOpts{Stream: nopStream{}}

			if delivery == "handler" {
				opts = operatorsvc.OpenOpts{Handler: nopHandler{}}
			}

			session, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, opts)
			require.NoError(t, err)

			changed, err := session.ApplyDelta(t.Context(), []string{"svc:a", "svc:b"}, nil)
			require.NoError(t, err)
			assert.True(t, changed)

			// a repeated add does not consume budget, a removal gives it back
			_, err = session.ApplyDelta(t.Context(), []string{"svc:a"}, []string{"svc:b"})
			require.NoError(t, err)

			_, err = session.ApplyDelta(t.Context(), []string{"svc:c"}, nil)
			require.NoError(t, err)
			assert.ElementsMatch(t, []string{"svc:a", "svc:c"}, svc.workers.ActionSet(worker.ID))

			_, err = session.ApplyDelta(t.Context(), []string{"svc:d"}, nil)
			assert.Equal(t, codes.ResourceExhausted, status.Code(err), err)
			assert.ElementsMatch(t, []string{"svc:a", "svc:c"}, svc.workers.ActionSet(worker.ID), "a refused delta is not applied")

			// a session at the cap can still give links back
			_, err = session.ApplyDelta(t.Context(), nil, []string{"svc:a"})
			require.NoError(t, err)

			_, err = session.ApplyDelta(t.Context(), []string{"svc:d"}, nil)
			require.NoError(t, err, "a removal frees budget for the next add")

			require.NoError(t, session.Close(t.Context()))
		})
	}
}

// A burst of deltas is folded into one scheduler reload per window, and a delta that changes
// nothing does not notify at all.
func TestSessionApplyDeltaThrottlesNotifications(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil, operatorsvc.WithNotifyInterval(100*time.Millisecond))
	op, worker := registeredOperator(t, svc, tenant)

	session, err := svc.OpenSession(t.Context(), tenant, op, worker.ID, operatorsvc.OpenOpts{Stream: nopStream{}})
	require.NoError(t, err)

	require.Equal(t, 1, svc.dispatcher.NotifyCount(), "opening the session notifies once")

	// deltas right after the opening notify are inside the throttle window, so they are
	// deferred and folded into one notification
	_, err = session.ApplyDelta(t.Context(), []string{"svc:a", "svc:b"}, nil)
	require.NoError(t, err)
	_, err = session.ApplyDelta(t.Context(), []string{"svc:c"}, []string{"svc:b"})
	require.NoError(t, err)

	assert.Equal(t, 1, svc.dispatcher.NotifyCount(), "deltas inside the window are not notified immediately")
	eventually(t, func() bool { return svc.dispatcher.NotifyCount() == 2 }, "deferred notify did not fire")

	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, 2, svc.dispatcher.NotifyCount(), "no further notify without further changes")

	changed, err := session.ApplyDelta(t.Context(), []string{"svc:a"}, []string{"svc:never"})
	require.NoError(t, err)
	assert.False(t, changed)
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, 2, svc.dispatcher.NotifyCount(), "a delta that changes nothing does not notify")

	_, err = session.ApplyDelta(t.Context(), nil, []string{"svc:a"})
	require.NoError(t, err)
	eventually(t, func() bool { return svc.dispatcher.NotifyCount() == 3 }, "delta outside the window did not notify")

	require.NoError(t, session.Close(t.Context(), operatorsvc.WithoutPause()))

	// the notifier is stopped with the session
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, 3, svc.dispatcher.NotifyCount())
}

// A session opens against a worker the caller has not read: the row is loaded so the pin is
// only written when it is wrong.
func TestOpenSessionLoadsWorkerWhenNotGiven(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)
	op, _ := registeredOperator(t, svc, tenant)

	reg, err := svc.Register(t.Context(), tenant, grpcRegisterOpts(op.Name))
	require.NoError(t, err)

	session, err := svc.OpenSession(t.Context(), tenant, op, reg.WorkerId, operatorsvc.OpenOpts{Handler: nopHandler{}})
	require.NoError(t, err)

	assert.Equal(t, svc.dispatcherId, svc.workers.DispatcherFor(reg.WorkerId))
	assert.NotContains(t, strings.Join(svc.workers.Ops(), ","), "pause")

	require.NoError(t, session.Close(t.Context(), operatorsvc.WithoutPause()))
}
