//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
)

func TestSlotCap(t *testing.T) {
	assert.Equal(t, 1, slotCap(nil))
	assert.Equal(t, 1, slotCap(map[string]int32{"a": -1}))
	assert.Equal(t, 30, slotCap(map[string]int32{"a": 10, "b": 20}))
}

// holdingSender installs a trigger handler that blocks until release is called, and reports
// on started each time a delivery reached it.
func holdingSender(env *testEnv, url string) (started chan struct{}, release func()) {
	started = make(chan struct{}, 16)
	gate := make(chan struct{})

	env.sender.handle(url, func(ctx context.Context, _ senderCall) (*safeclient.DeliveryResult, error) {
		started <- struct{}{}

		select {
		case <-gate:
			return &safeclient.DeliveryResult{StatusCode: http.StatusOK, BodyPrefix: []byte(`{}`)}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	return started, func() { close(gate) }
}

// A start that finds every slot taken waits for one rather than refusing the action: the
// wait ends with the caller's context, and a later start goes through once a delivery has
// finished. No event is reported for the start that waited in vain, so the engine's own
// retry of the assignment is the only consequence.
func TestHandleActionBlocksOnFullSlots(t *testing.T) {
	env := newTestEnv(t)
	env.r.cfg.DefaultSlots = 1
	env.r.cfg.DurableSlots = 0
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	env.addEndpoint(a)
	started, release := holdingSender(env, a.TriggerUrl)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	session := env.host.session(0)
	require.NotNil(t, session)
	require.Equal(t, map[string]int32{"default": 1, "durable": 0}, env.host.opens[0].opts.SlotConfig)

	first := startAction(a.Namespace, "svc:run")
	require.NoError(t, session.handler.HandleAction(context.Background(), first))
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	second := startAction(a.Namespace, "svc:run")
	err := session.handler.HandleAction(ctx, second)
	require.ErrorIs(t, err, context.DeadlineExceeded, "the start waited on the caller's context instead of being refused")
	assert.Len(t, env.sender.callsTo(a.TriggerUrl), 1, "the waiting start delivered nothing")
	assert.Equal(t, 1, env.r.InFlight(env.unit(a)))

	release()
	require.Eventually(t, func() bool { return env.r.InFlight(env.unit(a)) == 0 }, eventually, 10*time.Millisecond)

	require.NoError(t, session.handler.HandleAction(context.Background(), second), "a slot is free again")
	require.Eventually(t, func() bool { return len(session.eventTypes()) == 4 }, eventually, 10*time.Millisecond)

	assert.Equal(t, []contracts.StepActionEventType{
		contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED,
		contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED,
		contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED,
		contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED,
	}, session.eventTypes(), "only the two deliveries reported; the refused wait did not")
}

// A cancel never waits for a slot: it must reach a delivery that holds the last one.
func TestCancelDoesNotTakeASlot(t *testing.T) {
	env := newTestEnv(t)
	env.r.cfg.DefaultSlots = 1
	env.r.cfg.DurableSlots = 0
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	env.addEndpoint(a)
	started, release := holdingSender(env, a.TriggerUrl)
	defer release()

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	session := env.host.session(0)

	action := startAction(a.Namespace, "svc:run")
	session.deliver(t, action)
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	require.NoError(t, session.handler.HandleAction(ctx, &contracts.AssignedAction{
		ActionType:        contracts.ActionType_CANCEL_STEP_RUN,
		TaskRunExternalId: action.TaskRunExternalId,
	}))

	require.Eventually(t, func() bool { return env.r.InFlight(env.unit(a)) == 0 }, eventually, 10*time.Millisecond)
	assert.Contains(t, session.eventTypes(), contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED)
}

// Actions that reach a registration outside its life are answered with an error, so the host
// hands them elsewhere: before the open produced a session, the handler waits on the caller's
// context, then reports that there is no session; after the close, it refuses at once.
// Action types a serverless worker has no use for are dropped without error.
func TestHandleActionOutsideTheRegistrationsLife(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	env.addEndpoint(a)

	unopened := newRegistration(env.r, env.r.tenantFor(tenant), nil, 0, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	require.ErrorIs(t, unopened.HandleAction(ctx, startAction(a.Namespace, "svc:run")), context.DeadlineExceeded, "the handler waits for the open")

	unopened.open(nil)
	require.ErrorIs(t, unopened.HandleAction(context.Background(), startAction(a.Namespace, "svc:run")), errRegistrationNotOpen)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	session := env.host.session(0)
	require.NotNil(t, session)

	require.NoError(t, session.handler.HandleAction(context.Background(), &contracts.AssignedAction{ActionType: contracts.ActionType_START_GET_GROUP_KEY}))
	assert.Empty(t, session.eventTypes(), "an unsupported action type is dropped quietly")

	env.r.UnitsLost(context.Background(), []memrepo.Unit{env.unit(a)})
	require.Eventually(t, session.isClosed, eventually, 10*time.Millisecond)

	require.ErrorIs(t, session.handler.HandleAction(context.Background(), startAction(a.Namespace, "svc:run")), errRegistrationClosed)
	assert.Empty(t, env.sender.callsTo(a.TriggerUrl))
}

// Losing a tenant's last unit tears its registration down in order: the worker is paused
// first, while its delivery is still in flight, the delivery is then awaited, and only then
// is the session closed. The delivery completes normally rather than being aborted.
func TestTeardownPausesBeforeDraining(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	env.addEndpoint(a)
	started, release := holdingSender(env, a.TriggerUrl)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	session := env.host.session(0)
	reg := env.tenant(tenant).registration()
	require.NotNil(t, reg)

	session.deliver(t, startAction(a.Namespace, "svc:run"))
	<-started

	env.r.UnitsLost(context.Background(), []memrepo.Unit{env.unit(a)})

	require.Eventually(t, func() bool { return len(session.lifecycle()) == 1 }, eventually, 10*time.Millisecond)
	assert.Equal(t, []string{"pause"}, session.lifecycle(), "the pause is committed while the delivery is in flight")
	assert.False(t, session.isClosed())
	assert.Equal(t, 1, reg.inFlight(), "the drain waits on the delivery")
	assert.Equal(t, []contracts.StepActionEventType{contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED}, session.eventTypes())

	release()

	require.Eventually(t, session.isClosed, eventually, 10*time.Millisecond)
	assert.Equal(t, []string{"pause", "close"}, session.lifecycle())
	assert.Equal(t, []contracts.StepActionEventType{
		contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED,
		contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED,
	}, session.eventTypes(), "the delivery finished on its own terms before the close")
	assert.Eventually(t, func() bool { return len(env.host.releasedTenants()) == 1 }, eventually, 10*time.Millisecond)
}

// Shutdown tears every registration down the same way.
func TestShutdownPausesEveryRegistrationBeforeClosing(t *testing.T) {
	env := newTestEnv(t)

	a := healthyRow(endpointSpec{tenantId: uuid.New(), name: "a", actions: []string{"svc:a"}})
	b := healthyRow(endpointSpec{tenantId: uuid.New(), name: "b", actions: []string{"svc:b"}})
	env.addEndpoint(a)
	env.addEndpoint(b)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a), env.unit(b)})
	require.Equal(t, 2, env.host.openCount())

	env.r.Shutdown()

	for i := 0; i < 2; i++ {
		assert.Equal(t, []string{"pause", "close"}, env.host.session(i).lifecycle())
	}

	assert.Len(t, env.host.releasedTenants(), 2)
}
