//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

const eventually = 3 * time.Second

func healthyRow(spec endpointSpec) *sqlcv1.V1ServerlessEndpoint {
	spec.enabled = true
	spec.healthy = pgtype.Bool{Bool: true, Valid: true}

	row := newEndpointRow(spec)
	row.RegisteredActions = []string{prefixed(row.Namespace, spec.actions[0])}

	for _, a := range spec.actions[1:] {
		row.RegisteredActions = append(row.RegisteredActions, prefixed(row.Namespace, a))
	}

	return row
}

func TestUnitGainOpensRegistrationWithTenantUnion(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:a"}})
	b := healthyRow(endpointSpec{tenantId: tenant, name: "b", actions: []string{"svc:b", "svc:b2"}})
	env.addEndpoint(a)
	env.addEndpoint(b)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})

	require.Equal(t, 1, env.link.openCount())

	open := env.link.opens[0]
	assert.Equal(t, tenant, open.tenantId)
	assert.Equal(t, 0, open.shard)
	assert.Equal(t, sortedUnion(a.RegisteredActions, b.RegisteredActions), open.opts.Actions, "the union covers every enabled endpoint of the tenant, not only the owned ones")
	assert.Equal(t, map[string]int32{repository.SlotTypeDefault: 10, repository.SlotTypeDurable: 5}, open.opts.SlotConfig)
	assert.Equal(t, env.r.processId.String(), open.opts.Labels[workerLabelProcess])
	assert.Empty(t, open.opts.Workflows, "no workflow is known before the first healthcheck")

	// Both endpoints are on the owned unit, so both are polled once and nothing changes.
	require.Eventually(t, func() bool {
		return len(env.sender.callsTo(a.HealthcheckUrl)) == 1 && len(env.sender.callsTo(b.HealthcheckUrl)) == 1
	}, eventually, 10*time.Millisecond)

	call := env.sender.callsTo(a.HealthcheckUrl)[0]
	assert.Equal(t, a.ID.String(), call.headers.Get("X-Hatchet-Endpoint-Id"))
	assert.Contains(t, string(call.body), a.Namespace.String())

	assert.Empty(t, env.repo.ActionWrites(), "an unchanged action set is not rewritten")
	assert.Empty(t, env.repo.StatusWrites(), "a healthy endpoint that stays healthy writes nothing")
	assert.Equal(t, 0, env.link.reg(0).updateCount())
}

func TestUnitLostClosesRegistrationWithoutTouchingActions(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:a"}})
	env.addEndpoint(a)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	require.Eventually(t, func() bool { return len(env.sender.callsTo(a.HealthcheckUrl)) == 1 }, eventually, 10*time.Millisecond)

	reg := env.link.reg(0)
	require.NotNil(t, reg)

	env.r.UnitsLost(context.Background(), []memrepo.Unit{env.unit(a)})

	require.Eventually(t, reg.isClosed, eventually, 10*time.Millisecond)
	assert.Nil(t, env.poller(a), "the poller stopped")
	assert.Nil(t, env.tenant(tenant), "the tenant is forgotten with its last unit")
	assert.Eventually(t, func() bool { return len(env.link.releasedTenants()) == 1 }, eventually, 10*time.Millisecond)
	assert.Equal(t, 0, reg.updateCount())
	assert.Equal(t, 0, reg.putCount())
	assert.Empty(t, env.repo.ActionWrites())
	assert.Equal(t, 1, env.link.openCount())
}

func TestHealthcheckChangeUpdatesActionsAndEveryRegistration(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	// Two shards of one tenant, one endpoint each: two registrations.
	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", shard: 0, actions: []string{"svc:a"}})
	b := healthyRow(endpointSpec{tenantId: tenant, name: "b", shard: 1, actions: []string{"svc:b"}})
	env.addEndpoint(a)
	env.addEndpoint(b)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a), env.unit(b)})
	require.Equal(t, 2, env.link.openCount())

	regA := env.link.reg(0)
	regB := env.link.reg(1)

	require.Eventually(t, func() bool {
		return len(env.sender.callsTo(a.HealthcheckUrl)) == 1 && len(env.sender.callsTo(b.HealthcheckUrl)) == 1
	}, eventually, 10*time.Millisecond)

	// Endpoint a now advertises a workflow instead of the bare action.
	poller := env.poller(a)
	require.NotNil(t, poller)
	poller.stop()

	wf := &v1.CreateWorkflowVersionRequest{
		Name:  "echo",
		Tasks: []*v1.CreateTaskOpts{{ReadableId: "run", Action: "svc:echo"}},
	}

	env.sender.respond(a.HealthcheckUrl, http.StatusOK, healthcheckWithWorkflows(t, wf))
	poller.pollOnce(context.Background())

	wantA := []string{prefixed(a.Namespace, "svc:echo")}
	wantUnion := sortedUnion(wantA, b.RegisteredActions)

	require.Equal(t, 1, regA.putCount(), "the owner's registration puts the changed workflow")
	assert.Equal(t, prefixed(a.Namespace, "echo"), regA.puts[0].wf.Name)
	assert.Equal(t, wantUnion, regA.puts[0].actions, "put with the tenant's full action set")

	require.Len(t, env.repo.ActionWrites(), 1)
	assert.Equal(t, a.ID, env.repo.ActionWrites()[0].EndpointId)
	assert.Equal(t, wantA, env.repo.ActionWrites()[0].Actions)

	require.Equal(t, 1, regB.updateCount(), "the other registration of the tenant is updated")
	assert.Equal(t, wantUnion, regB.updates[0])
	assert.Equal(t, 0, regA.updateCount(), "the put already carried the set for the owner")
	assert.Equal(t, 0, regB.putCount())

	// The same response again changes nothing.
	poller.pollOnce(context.Background())
	assert.Equal(t, 1, regA.putCount())
	assert.Len(t, env.repo.ActionWrites(), 1)
	assert.Equal(t, 1, regB.updateCount())

	// Adding a second workflow puts only the new one.
	wf2 := &v1.CreateWorkflowVersionRequest{
		Name:  "other",
		Tasks: []*v1.CreateTaskOpts{{ReadableId: "run", Action: "svc:other"}},
	}

	env.sender.respond(a.HealthcheckUrl, http.StatusOK, healthcheckWithWorkflows(t, wf, wf2))
	poller.pollOnce(context.Background())

	require.Equal(t, 2, regA.putCount(), "the unchanged workflow is not re-put")
	assert.Equal(t, prefixed(a.Namespace, "other"), regA.puts[1].wf.Name)
	assert.Len(t, env.repo.ActionWrites(), 2)
	assert.Equal(t, 2, regB.updateCount())
	wantUnion = sortedUnion([]string{prefixed(a.Namespace, "svc:echo"), prefixed(a.Namespace, "svc:other")}, b.RegisteredActions)
	assert.Equal(t, wantUnion, regB.updates[1])

	// A fresh registration for the tenant opens with the known workflow.
	env.r.UnitsLost(context.Background(), []memrepo.Unit{env.unit(b)})
	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(b)})

	require.Equal(t, 3, env.link.openCount())
	reopened := env.link.opens[2]
	require.Len(t, reopened.opts.Workflows, 2)

	names := []string{reopened.opts.Workflows[0].Name, reopened.opts.Workflows[1].Name}
	assert.ElementsMatch(t, []string{prefixed(a.Namespace, "echo"), prefixed(a.Namespace, "other")}, names)
	assert.Equal(t, wantUnion, reopened.opts.Actions)
}

func TestEngineRejectedWorkflowMarksEndpoint(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:a"}})
	env.addEndpoint(a)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	require.Eventually(t, func() bool { return len(env.sender.callsTo(a.HealthcheckUrl)) == 1 }, eventually, 10*time.Millisecond)

	poller := env.poller(a)
	poller.stop()

	reg := env.link.reg(0)
	reg.mu.Lock()
	reg.putErr = assert.AnError
	reg.mu.Unlock()

	wf := &v1.CreateWorkflowVersionRequest{Name: "bad", Tasks: []*v1.CreateTaskOpts{{ReadableId: "t", Action: "svc:bad"}}}
	env.sender.respond(a.HealthcheckUrl, http.StatusOK, healthcheckWithWorkflows(t, wf))
	poller.pollOnce(context.Background())

	require.Len(t, env.repo.StatusWrites(), 1)
	assert.False(t, env.repo.StatusWrites()[0].Healthy)
	assert.Contains(t, *env.repo.StatusWrites()[0].Error, "engine rejected workflow")
	assert.Empty(t, env.repo.ActionWrites(), "registered_actions is not written for a rejected workflow")
}

func TestTokenlessTenant(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:a"}})
	env.addEndpoint(a)
	env.link.setOpenErr(link.ErrNoToken)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})

	require.Len(t, env.repo.StatusWrites(), 1, "status error written once")
	assert.False(t, env.repo.StatusWrites()[0].Healthy)
	assert.Equal(t, noTokenStatusError, *env.repo.StatusWrites()[0].Error)
	assert.Nil(t, env.poller(a), "nothing is polled without a token")
	assert.Nil(t, env.link.reg(0))

	// Later passes without a token neither rewrite the status nor poll.
	env.r.maintainOnce(context.Background())
	assert.Len(t, env.repo.StatusWrites(), 1)
	assert.Empty(t, env.sender.callsTo(a.HealthcheckUrl))
	assert.Equal(t, 2, env.link.openCount(), "open is retried")

	// The token appears: the registration opens, polling starts and health recovers.
	env.link.setOpenErr(nil)
	env.r.maintainOnce(context.Background())

	require.NotNil(t, env.link.reg(0))
	require.Eventually(t, func() bool { return len(env.sender.callsTo(a.HealthcheckUrl)) == 1 }, eventually, 10*time.Millisecond)
	require.Eventually(t, func() bool { return len(env.repo.StatusWrites()) == 2 }, eventually, 10*time.Millisecond)
	assert.True(t, env.repo.StatusWrites()[1].Healthy)
	assert.Nil(t, env.repo.StatusWrites()[1].Error)
}

func TestStatusWritesOnlyOnTransitions(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	// Health unknown at first: the first success writes healthy=true.
	a := newEndpointRow(endpointSpec{tenantId: tenant, name: "a", enabled: true})
	a.RegisteredActions = []string{prefixed(a.Namespace, "svc:a")}
	env.addEndpoint(a)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	require.Eventually(t, func() bool { return len(env.repo.StatusWrites()) == 1 }, eventually, 10*time.Millisecond)
	assert.True(t, env.repo.StatusWrites()[0].Healthy)

	poller := env.poller(a)
	require.NotNil(t, poller)
	poller.stop()

	env.sender.respond(a.HealthcheckUrl, http.StatusBadGateway, "")

	poller.pollOnce(context.Background())
	poller.pollOnce(context.Background())
	assert.Len(t, env.repo.StatusWrites(), 1, "two failures are not yet a transition")

	poller.pollOnce(context.Background())
	require.Len(t, env.repo.StatusWrites(), 2, "the third consecutive failure flips unhealthy")
	assert.False(t, env.repo.StatusWrites()[1].Healthy)
	assert.Equal(t, "healthcheck returned status 502", *env.repo.StatusWrites()[1].Error)

	poller.pollOnce(context.Background())
	poller.pollOnce(context.Background())
	assert.Len(t, env.repo.StatusWrites(), 2, "staying unhealthy writes nothing")

	env.sender.respond(a.HealthcheckUrl, http.StatusOK, healthcheckBody("svc:a"))

	poller.pollOnce(context.Background())
	require.Len(t, env.repo.StatusWrites(), 3, "one success flips back")
	assert.True(t, env.repo.StatusWrites()[2].Healthy)

	poller.pollOnce(context.Background())
	assert.Len(t, env.repo.StatusWrites(), 3)

	// Two failures then a success reset the counter without a write.
	env.sender.respond(a.HealthcheckUrl, http.StatusBadGateway, "")
	poller.pollOnce(context.Background())
	poller.pollOnce(context.Background())
	env.sender.respond(a.HealthcheckUrl, http.StatusOK, healthcheckBody("svc:a"))
	poller.pollOnce(context.Background())
	env.sender.respond(a.HealthcheckUrl, http.StatusBadGateway, "")
	poller.pollOnce(context.Background())
	poller.pollOnce(context.Background())
	assert.Len(t, env.repo.StatusWrites(), 3)
}

func TestDeliveryRoutesByNamespace(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	b := healthyRow(endpointSpec{tenantId: tenant, name: "b", actions: []string{"svc:run"}})
	env.addEndpoint(a)
	env.addEndpoint(b)
	env.sender.respond(a.TriggerUrl, http.StatusOK, `{"from":"a"}`)
	env.sender.respond(b.TriggerUrl, http.StatusOK, `{"from":"b"}`)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	reg := env.link.reg(0)
	require.NotNil(t, reg)

	// Same action name on two endpoints: the namespace prefix decides.
	reg.actions <- startAction(b.Namespace, "svc:run")

	require.Eventually(t, func() bool { return len(env.sender.callsTo(b.TriggerUrl)) == 1 }, eventually, 10*time.Millisecond)
	require.Eventually(t, func() bool { return len(reg.eventTypes()) == 2 }, eventually, 10*time.Millisecond)

	assert.Empty(t, env.sender.callsTo(a.TriggerUrl))
	assert.Equal(t, []contracts.StepActionEventType{
		contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED,
		contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED,
	}, reg.eventTypes())

	done := reg.lastEvent()
	assert.Equal(t, `{"from":"b"}`, done.EventPayload)
	assert.Equal(t, reg.workerId, done.WorkerId)
	assert.Equal(t, int32(1), *done.RetryCount)
	assert.Nil(t, done.ShouldNotRetry)

	// A permanent endpoint failure is reported as non-retryable.
	env.sender.respond(a.TriggerUrl, http.StatusBadRequest, `{"error":"bad input"}`)
	reg.actions <- startAction(a.Namespace, "svc:run")

	require.Eventually(t, func() bool { return len(reg.eventTypes()) == 4 }, eventually, 10*time.Millisecond)
	failed := reg.lastEvent()
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, failed.EventType)
	assert.Equal(t, "bad input", failed.EventPayload)
	assert.True(t, *failed.ShouldNotRetry)
}

func TestDeliveryRoutingMissAndDurable(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	env.addEndpoint(a)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	reg := env.link.reg(0)

	loads := env.repo.ListForTenantCalls()

	reg.actions <- startAction(uuid.New(), "svc:run")

	require.Eventually(t, func() bool { return len(reg.eventTypes()) == 1 }, eventually, 10*time.Millisecond)
	miss := reg.lastEvent()
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, miss.EventType)
	assert.Contains(t, miss.EventPayload, "endpoint not found for namespace")
	assert.False(t, *miss.ShouldNotRetry)
	assert.Equal(t, loads+1, env.repo.ListForTenantCalls(), "a miss reloads the tenant once")

	invocation := int32(0)
	durable := startAction(a.Namespace, "svc:run")
	durable.DurableTaskInvocationCount = &invocation
	reg.actions <- durable

	// STARTED goes out before the channel is opened, then the link's refusal is reported.
	require.Eventually(t, func() bool { return len(reg.eventTypes()) == 3 }, eventually, 10*time.Millisecond)
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED, reg.eventTypes()[1])
	ev := reg.lastEvent()
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, ev.EventType)
	assert.Equal(t, link.ErrDurableNotSupported.Error(), ev.EventPayload)
	assert.False(t, *ev.ShouldNotRetry)
	assert.Empty(t, env.sender.callsTo(a.TriggerUrl))
}

func TestCancelInFlightDelivery(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	env.addEndpoint(a)

	started := make(chan struct{}, 1)

	env.sender.handle(a.TriggerUrl, func(ctx context.Context, _ senderCall) (*safeclient.DeliveryResult, error) {
		started <- struct{}{}
		<-ctx.Done()

		return nil, ctx.Err()
	})

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	reg := env.link.reg(0)

	action := startAction(a.Namespace, "svc:run")
	reg.actions <- action

	select {
	case <-started:
	case <-time.After(eventually):
		t.Fatal("delivery never started")
	}

	assert.Equal(t, 1, env.r.InFlight(env.unit(a)))

	cancel := &contracts.AssignedAction{
		ActionType:        contracts.ActionType_CANCEL_STEP_RUN,
		TaskRunExternalId: action.TaskRunExternalId,
		TaskId:            action.TaskId,
		ActionId:          action.ActionId,
	}
	reg.actions <- cancel

	require.Eventually(t, func() bool { return env.r.InFlight(env.unit(a)) == 0 }, eventually, 10*time.Millisecond)

	// STARTED, then exactly one CANCELLED from the cancel handler; the interrupted delivery
	// reports nothing on its own.
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, []contracts.StepActionEventType{
		contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED,
		contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED,
	}, reg.eventTypes())
}

func TestDrainTimeoutAbortsDeliveryRetryably(t *testing.T) {
	env := newTestEnv(t)
	env.r.cfg.DrainTimeout = 50 * time.Millisecond
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	env.addEndpoint(a)

	started := make(chan struct{}, 1)

	env.sender.handle(a.TriggerUrl, func(ctx context.Context, _ senderCall) (*safeclient.DeliveryResult, error) {
		started <- struct{}{}
		<-ctx.Done()

		return nil, ctx.Err()
	})

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	reg := env.link.reg(0)
	reg.actions <- startAction(a.Namespace, "svc:run")

	<-started

	env.r.UnitsLost(context.Background(), []memrepo.Unit{env.unit(a)})

	require.Eventually(t, reg.isClosed, eventually, 10*time.Millisecond)
	require.Eventually(t, func() bool { return len(reg.eventTypes()) == 2 }, eventually, 10*time.Millisecond)

	ev := reg.lastEvent()
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, ev.EventType)
	assert.False(t, *ev.ShouldNotRetry)
	assert.Contains(t, ev.EventPayload, "operator shutting down")
}

func TestMaintainStartsPollerForNewEndpointAndStopsForDeleted(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:a"}})
	env.addEndpoint(a)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	reg := env.link.reg(0)

	// A new endpoint on the owned unit appears in the database.
	b := healthyRow(endpointSpec{tenantId: tenant, name: "b", actions: []string{"svc:b"}})
	env.addEndpoint(b)

	env.r.maintainOnce(context.Background())

	require.NotNil(t, env.poller(b))
	require.Eventually(t, func() bool { return len(env.sender.callsTo(b.HealthcheckUrl)) == 1 }, eventually, 10*time.Millisecond)
	require.Equal(t, 1, reg.updateCount(), "the union grew")
	assert.Equal(t, sortedUnion(a.RegisteredActions, b.RegisteredActions), reg.updates[0])

	// Deleting it is only visible to a full reload.
	env.repo.RemoveEndpoint(b.ID)
	env.r.maintainOnce(context.Background())
	assert.NotNil(t, env.poller(b))

	env.r.cfg.RoutingFullReloadInterval = 0
	env.r.maintainOnce(context.Background())

	assert.Nil(t, env.poller(b))
	require.Equal(t, 2, reg.updateCount())
	assert.Equal(t, a.RegisteredActions, reg.updates[1])
}
