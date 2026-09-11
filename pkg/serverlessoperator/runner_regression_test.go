//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/lease"
)

// failingEndpoints fails ListForTenant while fail is set, standing in for a database outage
// during the initial load of a gained tenant.
type failingEndpoints struct {
	repository.ServerlessEndpointRepository
	mu   sync.Mutex
	fail bool
}

func (r *failingEndpoints) setFail(fail bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.fail = fail
}

func (r *failingEndpoints) ListForTenant(ctx context.Context, id uuid.UUID) ([]*sqlcv1.V1ServerlessEndpoint, error) {
	r.mu.Lock()
	fail := r.fail
	r.mu.Unlock()

	if fail {
		return nil, errors.New("transient database outage")
	}

	return r.ServerlessEndpointRepository.ListForTenant(ctx, id)
}

type failingRepo struct {
	repository.ServerlessRepository
	ep *failingEndpoints
}

func (r *failingRepo) Endpoints() repository.ServerlessEndpointRepository { return r.ep }

// A unit whose tenant could not be loaded when it was gained must be served once the
// database recovers: the claim is kept, so nothing else will ever gain it again.
func TestClaimedUnitIsServedAfterInitialLoadFailure(t *testing.T) {
	env := newTestEnv(t)
	row := healthyRow(endpointSpec{tenantId: uuid.New(), name: "load-failure", actions: []string{"svc:run"}})
	env.addEndpoint(row)

	ep := &failingEndpoints{ServerlessEndpointRepository: env.repo.Endpoints(), fail: true}
	env.r.repo = &failingRepo{ServerlessRepository: env.repo, ep: ep}

	ls := lease.New(env.r.repo, env.r, lease.Config{ProcessId: env.r.processId}, env.r.l, lease.Hooks{})
	ctx := context.Background()

	require.NoError(t, ls.Heartbeat(ctx))
	require.NoError(t, ls.Tick(ctx))
	require.Len(t, ls.Owned(), 1, "the unit is claimed")
	require.Equal(t, 0, env.host.openCount(), "nothing opened while the load failed")

	ep.setFail(false)

	for i := 0; i < 3 && env.host.openCount() == 0; i++ {
		require.NoError(t, ls.Heartbeat(ctx))
		require.NoError(t, ls.Tick(ctx))
		env.r.maintainOnce(ctx)
	}

	assert.Equal(t, 1, env.host.openCount(), "database recovered but the claimed unit is still unserved: owned=%d tenants=%d", len(ls.Owned()), len(env.r.tenants))
	assert.NotNil(t, env.poller(row), "the unit's endpoint is polled after recovery")
}

// When invocation N+1 of a durable task starts before invocation N's delivery has cleaned up,
// N's cleanup must not remove N+1's record: a later cancel must still reach N+1.
func TestOlderInvocationFinishKeepsCurrentInvocation(t *testing.T) {
	env := newTestEnv(t)

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	fake := newFakeSession(nil, operator.Registration{WorkerId: uuid.New()})
	reg := &registration{r: env.r, session: fake, events: &eventSender{session: fake}, inflight: map[string]map[attemptKey]*inflightTask{}}

	id := uuid.New().String()
	oldKey := attemptKey{retry: 1, invocation: 1}
	currentKey := attemptKey{retry: 1, invocation: 2}
	old := &inflightTask{cancel: cancel1}
	current := &inflightTask{cancel: cancel2}

	// Both records are installed the way startDelivery installs them, N first.
	reg.inflight[id] = map[attemptKey]*inflightTask{oldKey: old, currentKey: current}
	require.Equal(t, 2, reg.inFlight(), "both attempts are accounted for")

	// N's deferred cleanup runs.
	reg.finish(id, oldKey, old)
	require.Error(t, ctx1.Err(), "the old invocation finished")
	assert.Equal(t, 1, reg.inFlight(), "the current invocation is still accounted for")

	// A cancel naming N+1 reaches it.
	invocation := int32(2)
	reg.cancelTask(&contracts.AssignedAction{TaskRunExternalId: id, ActionType: contracts.ActionType_CANCEL_STEP_RUN, RetryCount: 1, DurableTaskInvocationCount: &invocation})

	assert.Error(t, ctx2.Err(), "the older invocation's cleanup erased the current invocation: the current context is still live after CANCEL")
	assert.True(t, current.byEngine.Load())

	// A finish for a record that was replaced by a newer attempt of the same key is a no-op.
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel3()
	newer := &inflightTask{cancel: cancel3}
	reg.inflight[id][currentKey] = newer
	reg.finish(id, currentKey, current)
	assert.Equal(t, 1, reg.inFlight(), "a stale finish leaves the newer record alone")

	// A cancel that names no in-flight attempt cancels every attempt of the task.
	reg.cancelTask(&contracts.AssignedAction{TaskRunExternalId: id, ActionType: contracts.ActionType_CANCEL_STEP_RUN, RetryCount: 7})
	assert.Error(t, ctx3.Err())
}

// A duplicate assignment of an attempt already in flight does not start a second delivery.
func TestDuplicateAssignmentIsIgnored(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", actions: []string{"svc:run"}})
	env.addEndpoint(a)

	release := make(chan struct{})
	env.sender.handle(a.TriggerUrl, func(ctx context.Context, _ senderCall) (*safeclient.DeliveryResult, error) {
		select {
		case <-release:
		case <-ctx.Done():
		}

		return &safeclient.DeliveryResult{StatusCode: http.StatusOK, BodyPrefix: []byte(`{}`)}, nil
	})

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	fake := env.host.session(0)
	reg := env.tenant(tenant).registration()
	require.NotNil(t, reg)

	action := startAction(a.Namespace, "svc:run")
	fake.deliver(t, action)
	require.Eventually(t, func() bool { return len(env.sender.callsTo(a.TriggerUrl)) == 1 }, eventually, 10*time.Millisecond)

	fake.deliver(t, action)
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, reg.inFlight())
	assert.Len(t, env.sender.callsTo(a.TriggerUrl), 1, "the duplicate assignment did not deliver again")

	close(release)
	require.Eventually(t, func() bool { return reg.inFlight() == 0 }, eventually, 10*time.Millisecond)
}

// blockedOpen is a session whose OpenDurable reports the context it was given and blocks
// until that context ends.
type blockedOpen struct {
	operator.Session
	observed chan context.Context
}

func (r *blockedOpen) OpenDurable(ctx context.Context, _ uuid.UUID, _ int32) (operator.DurableChannel, error) {
	r.observed <- ctx
	<-ctx.Done()

	return nil, ctx.Err()
}

// The in-engine handshake of a durable invocation is bounded by the endpoint's request
// timeout like the rest of the invocation.
func TestDurableHandshakeCarriesRequestDeadline(t *testing.T) {
	env := newTestEnv(t)
	fake := newFakeSession(nil, operator.Registration{WorkerId: uuid.New()})
	blocking := &blockedOpen{Session: fake, observed: make(chan context.Context, 1)}
	reg := &registration{r: env.r, session: blocking, events: &eventSender{session: fake}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	invocation := int32(1)
	action := &contracts.AssignedAction{TaskRunExternalId: uuid.NewString(), DurableTaskInvocationCount: &invocation}

	go func() {
		defer close(done)
		reg.deliverDurable(ctx, &inflightTask{}, action, &cachedEndpoint{id: uuid.New(), namespace: uuid.New()}, &endpointConfig{requestTimeoutSeconds: 1, secret: "test-secret"}, time.Now())
	}()

	openCtx := <-blocking.observed
	deadline, bounded := openCtx.Deadline()
	cancel()
	<-done

	require.True(t, bounded, "requestTimeoutSeconds=1 but OpenDurable received a context without a deadline")
	assert.WithinDuration(t, time.Now().Add(time.Second), deadline, 2*time.Second)
}

// blockingHost blocks Open for one tenant until released, standing in for a hung engine.
type blockingHost struct {
	fakeHost
	blockTenant uuid.UUID
	release     chan struct{}
	entered     chan struct{}
}

func (h *blockingHost) Open(ctx context.Context, id operator.Identity, opts operator.OpenOpts) (operator.Session, error) {
	if id.TenantId == h.blockTenant {
		h.entered <- struct{}{}

		select {
		case <-h.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return h.fakeHost.Open(ctx, id, opts)
}

// One tenant's hung registration must not stall reconciliation of another tenant: the runner
// lock covers state transitions, not network work.
func TestHungOpenDoesNotBlockOtherTenants(t *testing.T) {
	env := newTestEnv(t)
	slow := uuid.New()
	fast := uuid.New()

	host := &blockingHost{blockTenant: slow, release: make(chan struct{}), entered: make(chan struct{}, 1)}
	env.r.host = host
	defer close(host.release)

	a := healthyRow(endpointSpec{tenantId: slow, name: "slow", actions: []string{"svc:a"}})
	b := healthyRow(endpointSpec{tenantId: fast, name: "fast", actions: []string{"svc:b"}})
	env.addEndpoint(a)
	env.addEndpoint(b)

	go env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a)})
	<-host.entered

	done := make(chan struct{})

	go func() {
		defer close(done)
		env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(b)})
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("gaining a unit of another tenant waited behind a hung registration open")
	}

	assert.NotNil(t, env.poller(b), "the fast tenant is being served")
}

// A tenant with many slow endpoints must not hold every healthcheck slot: a per-tenant bound
// inside the process-wide limit leaves room for other tenants.
func TestOneTenantCannotStarveHealthchecks(t *testing.T) {
	env := newTestEnv(t)
	hog := uuid.New()
	other := uuid.New()

	// Four slots process-wide, two per tenant; the hog's healthchecks hang far longer than
	// the wait below.
	env.r.cfg.HealthcheckTimeout = 30 * time.Second
	env.r.cfg.HealthcheckTenantConcurrency = 2

	stuck := make(chan struct{})
	defer close(stuck)

	rows := make([]*sqlcv1.V1ServerlessEndpoint, 0, 6)

	for i := 0; i < 6; i++ {
		row := healthyRow(endpointSpec{tenantId: hog, name: fmt.Sprintf("hog-%d", i), actions: []string{"svc:a"}})
		env.addEndpoint(row)
		env.sender.handle(row.HealthcheckUrl, func(ctx context.Context, _ senderCall) (*safeclient.DeliveryResult, error) {
			select {
			case <-stuck:
			case <-ctx.Done():
			}

			return &safeclient.DeliveryResult{StatusCode: http.StatusServiceUnavailable}, nil
		})
		rows = append(rows, row)
	}

	b := healthyRow(endpointSpec{tenantId: other, name: "other", actions: []string{"svc:b"}})
	env.addEndpoint(b)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(rows[0])})

	// The hog's pollers are all blocked inside their healthchecks.
	require.Eventually(t, func() bool { return len(env.r.hcSem) >= 2 }, eventually, 10*time.Millisecond)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(b)})

	assert.Eventually(t, func() bool { return len(env.sender.callsTo(b.HealthcheckUrl)) == 1 }, eventually, 10*time.Millisecond,
		"the other tenant's healthcheck never ran while one tenant held every slot")
}

var testLimits = catalogLimits{maxWorkflows: DefaultMaxWorkflowsPerEndpoint, maxActions: DefaultMaxActionsPerEndpoint}

// Every advertised action, derived or listed, must pass the engine's action validation
// before it can enter the shared tenant union.
func TestHealthcheckRejectsInvalidActions(t *testing.T) {
	ns := uuid.New()

	_, err := parseHealthcheckResponse([]byte(`{"actions":["svc:"]}`), ns, testLimits)
	assert.Error(t, err, "an action with an empty verb must be rejected")

	_, err = parseHealthcheckResponse([]byte(`{"actions":[":run"]}`), ns, testLimits)
	assert.Error(t, err, "an action with an empty service must be rejected")

	_, err = parseHealthcheckResponse([]byte(`{"workflows":[{"name":"w","tasks":[{"readableId":"t","action":"svc:"}]}]}`), ns, testLimits)
	assert.Error(t, err, "a derived action with an empty verb must be rejected")

	res, err := parseHealthcheckResponse([]byte(`{"actions":["Svc:Run"]}`), ns, testLimits)
	require.NoError(t, err)
	assert.Equal(t, []string{prefixed(ns, "svc:run")}, res.actions)
}

// A byte-bounded healthcheck must not admit an unbounded number of workflows or actions.
func TestHealthcheckCatalogIsCapped(t *testing.T) {
	ns := uuid.New()

	workflows := make([]map[string]any, 10000)

	for i := range workflows {
		workflows[i] = map[string]any{"name": fmt.Sprintf("flow-%d", i), "tasks": []map[string]string{{"readableId": "step", "action": fmt.Sprintf("svc:a%d", i)}}}
	}

	body, err := json.Marshal(map[string]any{"workflows": workflows})
	require.NoError(t, err)

	_, err = parseHealthcheckResponse(body, ns, testLimits)
	require.Error(t, err, "10000 workflows must be refused")
	assert.Contains(t, err.Error(), "workflows")

	actions := make([]string, 10000)

	for i := range actions {
		actions[i] = fmt.Sprintf("svc:a%d", i)
	}

	body, err = json.Marshal(map[string]any{"actions": actions})
	require.NoError(t, err)

	_, err = parseHealthcheckResponse(body, ns, testLimits)
	require.Error(t, err, "10000 actions must be refused")
	assert.Contains(t, err.Error(), "actions")
}

// A process owning several units of one tenant holds one registration for the tenant, which
// outlives any one unit and closes with the last.
func TestUnitsOfOneTenantShareOneRegistration(t *testing.T) {
	env := newTestEnv(t)
	tenant := uuid.New()

	a := healthyRow(endpointSpec{tenantId: tenant, name: "a", shard: 0, actions: []string{"svc:a"}})
	b := healthyRow(endpointSpec{tenantId: tenant, name: "b", shard: 1, actions: []string{"svc:b"}})
	env.addEndpoint(a)
	env.addEndpoint(b)

	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(a), env.unit(b)})
	require.Equal(t, 1, env.host.openCount(), "two units of one tenant open one registration")
	assert.Equal(t, sortedUnion(a.RegisteredActions, b.RegisteredActions), env.host.opens[0].opts.Actions)

	require.Eventually(t, func() bool {
		return len(env.sender.callsTo(a.HealthcheckUrl)) == 1 && len(env.sender.callsTo(b.HealthcheckUrl)) == 1
	}, eventually, 10*time.Millisecond)

	reg := env.host.session(0)

	// Losing one unit stops its poller and keeps the registration.
	env.r.UnitsLost(context.Background(), []memrepo.Unit{env.unit(b)})
	assert.Nil(t, env.poller(b))
	assert.NotNil(t, env.poller(a))
	assert.False(t, reg.isClosed())
	assert.Equal(t, 1, env.host.openCount())

	// Regaining it reuses the registration.
	env.r.UnitsGained(context.Background(), []memrepo.Unit{env.unit(b)})
	assert.NotNil(t, env.poller(b))
	assert.Equal(t, 1, env.host.openCount())

	// Losing the last unit closes it and releases the tenant.
	env.r.UnitsLost(context.Background(), []memrepo.Unit{env.unit(a), env.unit(b)})
	require.Eventually(t, reg.isClosed, eventually, 10*time.Millisecond)
	assert.Nil(t, env.tenant(tenant))
	assert.Eventually(t, func() bool { return len(env.host.releasedTenants()) == 1 }, eventually, 10*time.Millisecond)
}
