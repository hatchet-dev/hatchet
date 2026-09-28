//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/hostgrpc"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
)

const eventually = 3 * time.Second

func healthyRow(spec endpointSpec) *sqlcv1.V1ServerlessEndpoint {
	spec.enabled = true
	spec.healthy = pgtype.Bool{Bool: true, Valid: true}

	row := newEndpointRow(spec)
	row.RegisteredActions = append([]string{}, spec.actions...)

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

	require.Equal(t, 1, env.host.openCount())

	open := env.host.opens[0]
	assert.Equal(t, tenant, open.id.TenantId)
	assert.Equal(t, sortedUnion(a.RegisteredActions, b.RegisteredActions), open.opts.Actions, "the union covers every enabled endpoint of the tenant, not only the owned ones")
	assert.Equal(t, map[string]int32{repository.SlotTypeDefault: 10, repository.SlotTypeDurable: 5}, open.opts.SlotConfig)
	assert.Equal(t, env.r.processId.String(), open.opts.Labels[workerLabelProcess])

	// Both endpoints are on the owned unit, so both are polled once and nothing changes.
	require.Eventually(t, func() bool {
		return len(env.sender.callsTo(a.HealthcheckUrl)) == 1 && len(env.sender.callsTo(b.HealthcheckUrl)) == 1
	}, eventually, 10*time.Millisecond)

	call := env.sender.callsTo(a.HealthcheckUrl)[0]
	assert.Equal(t, a.ID.String(), call.headers.Get("X-Hatchet-Endpoint-Id"))
	assert.Contains(t, string(call.body), a.ID.String())

	assert.Empty(t, env.repo.ActionWrites(), "an unchanged action set is not rewritten")
	assert.Empty(t, env.repo.StatusWrites(), "a healthy endpoint that stays healthy writes nothing")
	assert.Equal(t, 0, env.host.session(0).deltaCount(), "the open carried the union; nothing to push")
}

func TestUnitLostClosesRegistrationWithoutTouchingActions(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:a"}})
	env.addEndpoint(a)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	require.Eventually(t, func() bool { return len(env.sender.callsTo(a.HealthcheckUrl)) == 1 }, eventually, 10*time.Millisecond)

	reg := env.host.session(0)
	require.NotNil(t, reg)

	env.r.UnitsLost(context.Background(), []memrepo.Unit{env.unit(a)})

	require.Eventually(t, reg.isClosed, eventually, 10*time.Millisecond)
	assert.Nil(t, env.poller(a), "the poller stopped")
	assert.Nil(t, env.tenant(tenant), "the tenant is forgotten with its last unit")
	assert.Eventually(t, func() bool { return len(env.host.releasedTenants()) == 1 }, eventually, 10*time.Millisecond)
	assert.Equal(t, 0, reg.deltaCount())
	assert.Equal(t, 0, reg.putCount())
	assert.Empty(t, env.repo.ActionWrites())
	assert.Equal(t, 1, env.host.openCount())
}

func TestHealthcheckChangeUpdatesActionsAndRegistration(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	// Two shards of one tenant, one endpoint each, served by the tenant's one registration.
	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", shard: 0, actions: []string{"svc:a"}})
	b := healthyRow(endpointSpec{tenantId: tenant, name: "b", shard: 1, actions: []string{"svc:b"}})
	env.addEndpoint(a)
	env.addEndpoint(b)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a), env.unit(b)})
	require.Equal(t, 1, env.host.openCount())

	reg := env.host.session(0)

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

	wantA := []string{"svc:echo"}

	require.Equal(t, 1, reg.putCount(), "the registration puts the changed workflow")
	assert.Equal(t, "echo", reg.puts[0].Name)

	require.Len(t, env.repo.ActionWrites(), 1)
	assert.Equal(t, a.ID, env.repo.ActionWrites()[0].EndpointId)
	assert.Equal(t, wantA, env.repo.ActionWrites()[0].Actions)

	// The registration receives the delta: the workflow's action in, the bare action out,
	// flushed once.
	assert.Equal(t, wantA, reg.added())
	assert.Equal(t, a.RegisteredActions, reg.removed())
	assert.Equal(t, 1, reg.flushCount())

	// The same response again changes nothing.
	poller.pollOnce(context.Background())
	assert.Equal(t, 1, reg.putCount())
	assert.Len(t, env.repo.ActionWrites(), 1)
	assert.Equal(t, 1, reg.flushCount())

	// Adding a second workflow puts only the new one and adds only its action.
	wf2 := &v1.CreateWorkflowVersionRequest{
		Name:  "other",
		Tasks: []*v1.CreateTaskOpts{{ReadableId: "run", Action: "svc:other"}},
	}

	env.sender.respond(a.HealthcheckUrl, http.StatusOK, healthcheckWithWorkflows(t, wf, wf2))
	poller.pollOnce(context.Background())

	require.Equal(t, 2, reg.putCount(), "the unchanged workflow is not re-put")
	assert.Equal(t, "other", reg.puts[1].Name)
	assert.Len(t, env.repo.ActionWrites(), 2)

	wantA = []string{"svc:echo", "svc:other"}
	wantUnion := sortedUnion(wantA, b.RegisteredActions)

	assert.Equal(t, wantA, reg.added(), "only the new action is added")
	assert.Equal(t, a.RegisteredActions, reg.removed(), "nothing else is removed")
	assert.Equal(t, 2, reg.flushCount())

	// A fresh registration for the tenant opens with the current union and needs no delta.
	env.r.UnitsLost(context.Background(), []memrepo.Unit{env.unit(a), env.unit(b)})
	require.Eventually(t, reg.isClosed, eventually, 10*time.Millisecond)
	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(b)})

	require.Equal(t, 2, env.host.openCount())
	reopened := env.host.opens[1]
	assert.Equal(t, wantUnion, reopened.opts.Actions)
	assert.Equal(t, 0, env.host.session(1).deltaCount())
}

func TestFailedDeltaIsRetriedOnNextPoll(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:a"}})
	env.addEndpoint(a)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	require.Eventually(t, func() bool { return len(env.sender.callsTo(a.HealthcheckUrl)) == 1 }, eventually, 10*time.Millisecond)

	poller := env.poller(a)
	poller.stop()

	reg := env.host.session(0)
	reg.mu.Lock()
	reg.deltaErr = assert.AnError
	reg.mu.Unlock()

	env.sender.respond(a.HealthcheckUrl, http.StatusOK, healthcheckBody("svc:a", "svc:b"))
	poller.pollOnce(context.Background())

	require.Len(t, env.repo.StatusWrites(), 1, "a delta the engine refused marks the endpoint")
	assert.False(t, env.repo.StatusWrites()[0].Healthy)
	assert.Contains(t, *env.repo.StatusWrites()[0].Error, "could not add actions")
	require.Len(t, env.repo.ActionWrites(), 1, "the row is written before the delta, so other processes see the set the endpoint serves")

	union, _ := env.tenant(tenant).cache.ActionUnion()
	assert.Equal(t, []string{"svc:a"}, union, "the cached union is restored while the engine does not have the actions")

	// The engine accepts again: the same delta is pushed without another write and the
	// endpoint recovers.
	reg.mu.Lock()
	reg.deltaErr = nil
	reg.mu.Unlock()

	poller.pollOnce(context.Background())

	assert.Equal(t, []string{"svc:b"}, reg.added())
	assert.Empty(t, reg.removed())
	assert.Equal(t, 1, reg.flushCount())
	require.Len(t, env.repo.ActionWrites(), 1)
	require.Len(t, env.repo.StatusWrites(), 2)
	assert.True(t, env.repo.StatusWrites()[1].Healthy)
}

// A registered_actions write that fails leaves the engine, the cache and the row where they
// were: the session gets no delta and the whole change is retried on the next poll.
func TestFailedActionWriteChangesNothing(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:a"}})
	env.addEndpoint(a)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	require.Eventually(t, func() bool { return len(env.sender.callsTo(a.HealthcheckUrl)) == 1 }, eventually, 10*time.Millisecond)

	poller := env.poller(a)
	poller.stop()

	reg := env.host.session(0)

	env.repo.SetFailWrites(assert.AnError)
	env.sender.respond(a.HealthcheckUrl, http.StatusOK, healthcheckBody("svc:a", "svc:b"))
	poller.pollOnce(context.Background())

	assert.Equal(t, 0, reg.deltaCount(), "the session sees no delta while the row is not written")
	assert.Empty(t, env.repo.ActionWrites())

	union, _ := env.tenant(tenant).cache.ActionUnion()
	assert.Equal(t, []string{"svc:a"}, union, "the cached union is unchanged")

	// The write succeeds on the next poll: the row, the cache and the session all move.
	env.repo.SetFailWrites(nil)
	poller.pollOnce(context.Background())

	require.Len(t, env.repo.ActionWrites(), 1)
	assert.Equal(t, []string{"svc:a", "svc:b"}, env.repo.ActionWrites()[0].Actions)
	assert.Equal(t, []string{"svc:b"}, reg.added())
	assert.Equal(t, 1, reg.flushCount())

	union, _ = env.tenant(tenant).cache.ActionUnion()
	assert.Equal(t, []string{"svc:a", "svc:b"}, union)
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

	reg := env.host.session(0)
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
	env.host.setOpenErr(hostgrpc.ErrNoToken)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})

	require.Len(t, env.repo.StatusWrites(), 1, "status error written once")
	assert.False(t, env.repo.StatusWrites()[0].Healthy)
	assert.Equal(t, noTokenStatusError, *env.repo.StatusWrites()[0].Error)
	assert.Nil(t, env.poller(a), "nothing is polled without a token")
	assert.Nil(t, env.host.session(0))

	// Later passes without a token neither rewrite the status nor poll.
	env.r.maintainOnce(context.Background())
	assert.Len(t, env.repo.StatusWrites(), 1)
	assert.Empty(t, env.sender.callsTo(a.HealthcheckUrl))
	assert.Equal(t, 2, env.host.openCount(), "open is retried")

	// The token appears: the registration opens, polling starts and health recovers.
	env.host.setOpenErr(nil)
	env.r.maintainOnce(context.Background())

	require.NotNil(t, env.host.session(0))
	require.Eventually(t, func() bool { return len(env.sender.callsTo(a.HealthcheckUrl)) == 1 }, eventually, 10*time.Millisecond)
	require.Eventually(t, func() bool { return len(env.repo.StatusWrites()) == 2 }, eventually, 10*time.Millisecond)
	assert.True(t, env.repo.StatusWrites()[1].Healthy)
	assert.Nil(t, env.repo.StatusWrites()[1].Error)
}

// A host that gives up on a session (the gRPC host after a token source stopped having a
// token, a tenant mismatch) reports it through Done and Err; the runner opens a new
// registration for the tenant instead of leaving it registered and unserved.
func TestRegistrationReopensAfterHostGivesUp(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:a"}})
	env.addEndpoint(a)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})

	first := env.host.session(0)
	require.NotNil(t, first)

	first.fail(errors.New("token revoked"))

	require.Eventually(t, func() bool { return env.host.openCount() == 2 }, eventually, 10*time.Millisecond)
	require.Eventually(t, first.isClosed, eventually, 10*time.Millisecond)

	second := env.host.session(1)
	require.NotNil(t, second)
	require.Eventually(t, func() bool {
		ts := env.tenant(tenant)
		return ts != nil && ts.registration() != nil && ts.registration().session == second
	}, eventually, 10*time.Millisecond)

	assert.Equal(t, sortedUnion(a.RegisteredActions), env.host.opens[1].opts.Actions, "the new session opens with the tenant's current union")
	assert.Empty(t, env.host.releasedTenants(), "the tenant is still served; nothing is released on the host")

	// A registration closed by the runner itself is not reopened.
	env.r.UnitsLost(context.Background(), []memrepo.Unit{env.unit(a)})
	require.Eventually(t, second.isClosed, eventually, 10*time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, env.host.openCount())
}

func TestStatusWritesOnlyOnTransitions(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	// Health unknown at first: the first success writes healthy=true.
	a := newEndpointRow(endpointSpec{tenantId: tenant, name: "a", enabled: true})
	a.RegisteredActions = []string{"svc:a"}
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

func TestDeliveryRoutesToAnEndpointServingTheAction(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	b := healthyRow(endpointSpec{tenantId: tenant, name: "b", actions: []string{"svc:other"}})
	env.addEndpoint(a)
	env.addEndpoint(b)
	env.sender.respond(a.TriggerUrl, http.StatusOK, `{"from":"a"}`)
	env.sender.respond(b.TriggerUrl, http.StatusOK, `{"from":"b"}`)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	reg := env.host.session(0)
	require.NotNil(t, reg)

	// Each endpoint serves its own action: the delivery goes to the one advertising it.
	reg.deliver(t, startAction("svc:other"))

	require.Eventually(t, func() bool { return len(env.sender.callsTo(b.TriggerUrl)) == 1 }, eventually, 10*time.Millisecond)
	require.Eventually(t, func() bool { return len(reg.eventTypes()) == 2 }, eventually, 10*time.Millisecond)

	assert.Empty(t, env.sender.callsTo(a.TriggerUrl))
	assert.Equal(t, []contracts.StepActionEventType{
		contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED,
		contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED,
	}, reg.eventTypes())

	done := reg.lastEvent()
	assert.Equal(t, `{"from":"b"}`, done.EventPayload)
	assert.Equal(t, reg.workerId(), done.WorkerId)
	assert.Equal(t, int32(1), *done.RetryCount)
	assert.Nil(t, done.ShouldNotRetry)

	// A permanent endpoint failure is reported as non-retryable.
	env.sender.respond(a.TriggerUrl, http.StatusBadRequest, `{"error":"bad input"}`)
	reg.deliver(t, startAction("svc:run"))

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
	reg := env.host.session(0)

	loads := env.repo.ListForTenantCalls()
	since := env.repo.ListSinceCalls()

	reg.deliver(t, startAction("svc:missing"))

	require.Eventually(t, func() bool { return len(reg.eventTypes()) == 1 }, eventually, 10*time.Millisecond)
	miss := reg.lastEvent()
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, miss.EventType)
	assert.Contains(t, miss.EventPayload, errEndpointNotFound.Error())
	assert.False(t, *miss.ShouldNotRetry)
	assert.Equal(t, loads, env.repo.ListForTenantCalls(), "a miss does not reload the tenant")
	assert.Equal(t, since, env.repo.ListSinceCalls(), "a miss looks nothing up")

	invocation := int32(0)
	durable := startAction("svc:run")
	durable.DurableTaskInvocationCount = &invocation
	reg.deliver(t, durable)

	// STARTED goes out before the channel is opened, then the host's refusal is reported.
	require.Eventually(t, func() bool { return len(reg.eventTypes()) == 3 }, eventually, 10*time.Millisecond)
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED, reg.eventTypes()[1])
	ev := reg.lastEvent()
	assert.Equal(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, ev.EventType)
	assert.Equal(t, operator.ErrNotSupported.Error(), ev.EventPayload)
	assert.False(t, *ev.ShouldNotRetry)
	assert.Empty(t, env.sender.callsTo(a.TriggerUrl))
}

// Two endpoints of one tenant declaring the same action both serve it, like two workers:
// deliveries of the action spread over both and the registration advertises it once.
func TestDeliverySpreadsASharedActionOverItsEndpoints(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	b := healthyRow(endpointSpec{tenantId: tenant, name: "b", actions: []string{"svc:run"}})
	env.addEndpoint(a)
	env.addEndpoint(b)
	env.sender.respond(a.TriggerUrl, http.StatusOK, `{"from":"a"}`)
	env.sender.respond(b.TriggerUrl, http.StatusOK, `{"from":"b"}`)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	reg := env.host.session(0)
	require.NotNil(t, reg)
	assert.Equal(t, []string{"svc:run"}, env.host.opens[0].opts.Actions, "the union names the shared action once")

	const deliveries = 40

	for i := 0; i < deliveries; i++ {
		reg.deliver(t, startAction("svc:run"))
	}

	require.Eventually(t, func() bool {
		return len(env.sender.callsTo(a.TriggerUrl))+len(env.sender.callsTo(b.TriggerUrl)) == deliveries
	}, eventually, 10*time.Millisecond)
	require.Eventually(t, func() bool { return len(reg.eventTypes()) == 2*deliveries }, eventually, 10*time.Millisecond)

	assert.NotEmpty(t, env.sender.callsTo(a.TriggerUrl), "a serves some of the deliveries")
	assert.NotEmpty(t, env.sender.callsTo(b.TriggerUrl), "b serves some of the deliveries")

	for _, ev := range reg.eventTypes() {
		assert.NotEqual(t, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, ev)
	}
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
	reg := env.host.session(0)

	action := startAction("svc:run")
	reg.deliver(t, action)

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
	reg.deliver(t, cancel)

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
	reg := env.host.session(0)
	reg.deliver(t, startAction("svc:run"))

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
	reg := env.host.session(0)

	// A new endpoint on the owned unit appears in the database.
	b := healthyRow(endpointSpec{tenantId: tenant, name: "b", actions: []string{"svc:b"}})
	env.addEndpoint(b)

	env.r.maintainOnce(context.Background())

	require.NotNil(t, env.poller(b))
	require.Eventually(t, func() bool { return len(env.sender.callsTo(b.HealthcheckUrl)) == 1 }, eventually, 10*time.Millisecond)
	require.Equal(t, 1, reg.deltaCount(), "the union grew")
	assert.Equal(t, b.RegisteredActions, reg.added())
	assert.Equal(t, 1, reg.flushCount())

	// Deleting it is visible to the next incremental refresh: the poller stops and the
	// action leaves the union without waiting for a reconcile.
	env.repo.RemoveEndpoint(b.ID)
	env.r.maintainOnce(context.Background())

	assert.Nil(t, env.poller(b))
	require.Equal(t, 2, reg.deltaCount())
	assert.Equal(t, b.RegisteredActions, reg.removed())
	assert.Equal(t, 2, reg.flushCount())

	// A reconcile after the purge has nothing left to drop.
	_, err := env.repo.Endpoints().PurgeDeleted(context.Background(), time.Now().Add(time.Second))
	require.NoError(t, err)

	env.r.cfg.RoutingFullReloadInterval = 0
	env.r.maintainOnce(context.Background())

	assert.Nil(t, env.poller(b))
	assert.Equal(t, 2, reg.deltaCount())
}
