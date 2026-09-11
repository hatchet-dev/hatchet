//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
)

func TestApplyNamespace(t *testing.T) {
	ns := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	wf := &v1.CreateWorkflowVersionRequest{
		Name:          "echo",
		EventTriggers: []string{"user:created"},
		CronTriggers:  []string{"*/5 * * * *"},
		Tasks: []*v1.CreateTaskOpts{
			{ReadableId: "step1", Action: "Svc:Run"},
			{ReadableId: "step2", Action: "svc:other:sub", Parents: []string{"step1"}},
		},
		OnFailureTask: &v1.CreateTaskOpts{ReadableId: "fail", Action: "svc:fail"},
	}

	out, err := applyNamespace(wf, ns)
	require.NoError(t, err)

	prefix := ns.String() + "_"

	assert.Equal(t, prefix+"echo", out.Name)
	assert.Equal(t, []string{prefix + "user:created"}, out.EventTriggers)
	assert.Equal(t, []string{"*/5 * * * *"}, out.CronTriggers, "cron triggers are expressions, not names")
	assert.Equal(t, prefix+"svc:run", out.Tasks[0].Action, "service and verb are normalized like the engine does")
	assert.Equal(t, prefix+"svc:other:sub", out.Tasks[1].Action)
	assert.Equal(t, prefix+"svc:fail", out.OnFailureTask.Action)

	// Deep copy: the input is untouched and applying twice is idempotent.
	assert.Equal(t, "echo", wf.Name)
	assert.Equal(t, "Svc:Run", wf.Tasks[0].Action)

	again, err := applyNamespace(out, ns)
	require.NoError(t, err)
	assert.Equal(t, out.Name, again.Name)
	assert.Equal(t, out.Tasks[0].Action, again.Tasks[0].Action)

	actions, err := actionsForWorkflow(out)
	require.NoError(t, err)
	assert.Equal(t, []string{prefix + "svc:run", prefix + "svc:other:sub", prefix + "svc:fail"}, actions)
}

func TestApplyNamespaceRejectsBadActions(t *testing.T) {
	ns := uuid.New()

	_, err := applyNamespace(&v1.CreateWorkflowVersionRequest{Name: "x", Tasks: []*v1.CreateTaskOpts{{Action: ""}}}, ns)
	assert.Error(t, err)

	_, err = applyNamespace(&v1.CreateWorkflowVersionRequest{Name: "x", Tasks: []*v1.CreateTaskOpts{{Action: "noverb"}}}, ns)
	assert.Error(t, err)

	_, err = applyNamespace(nil, ns)
	assert.Error(t, err)
}

func TestParseNamespace(t *testing.T) {
	ns := uuid.New()

	got, ok := ParseNamespace(ns.String() + "_svc:run")
	require.True(t, ok)
	assert.Equal(t, ns, got)

	_, ok = ParseNamespace("svc:run")
	assert.False(t, ok)

	_, ok = ParseNamespace(ns.String() + ":svc:run")
	assert.False(t, ok, "wrong separator")

	_, ok = ParseNamespace("not-a-uuid-but-36-chars-long-string_svc:run")
	assert.False(t, ok)

	_, ok = ParseNamespace(ns.String())
	assert.False(t, ok, "prefix alone")
}

func TestRoutingCacheRouteAndMissRefresh(t *testing.T) {
	repo := memrepo.New()
	tenant := uuid.New()
	l := zerolog.Nop()

	a := newEndpointRow(endpointSpec{tenantId: tenant, name: "a", enabled: true, actions: nil})
	a.RegisteredActions = []string{prefixed(a.Namespace, "svc:a")}
	repo.AddEndpoint(a)

	cache := newRoutingCache(tenant, repo.Endpoints(), fakeEnc{}, &l)
	require.NoError(t, cache.Load(context.Background()))
	require.Equal(t, 1, repo.ListForTenantCalls())

	ep, cfg, err := cache.Route(context.Background(), prefixed(a.Namespace, "svc:a"))
	require.NoError(t, err)
	assert.Equal(t, a.ID, ep.id)
	assert.Equal(t, "secret-a", cfg.secret, "secret is decrypted once at load")
	assert.Equal(t, a.TriggerUrl, cfg.triggerUrl)
	assert.Equal(t, 1, repo.ListForTenantCalls(), "a hit does not reload")

	// An unknown namespace is looked up on its own, never by reloading the tenant, and a
	// namespace found absent is not looked up again within the negative TTL.
	b := newEndpointRow(endpointSpec{tenantId: tenant, name: "b", enabled: true})
	b.RegisteredActions = []string{prefixed(b.Namespace, "svc:b")}

	_, _, err = cache.Route(context.Background(), prefixed(b.Namespace, "svc:b"))
	require.ErrorIs(t, err, errEndpointNotFound)
	assert.Equal(t, 1, repo.ListForTenantCalls(), "a miss does not reload the tenant")
	assert.Equal(t, 1, repo.ByNamespaceCalls())

	_, _, err = cache.Route(context.Background(), prefixed(b.Namespace, "svc:b"))
	require.ErrorIs(t, err, errEndpointNotFound)
	assert.Equal(t, 1, repo.ByNamespaceCalls(), "an absent namespace is remembered")

	// Once the endpoint exists and the negative entry has expired, the lookup finds it.
	repo.AddEndpoint(b)
	cache.missMu.Lock()
	delete(cache.missed, b.Namespace)
	cache.missMu.Unlock()

	ep, _, err = cache.Route(context.Background(), prefixed(b.Namespace, "svc:b"))
	require.NoError(t, err)
	assert.Equal(t, b.ID, ep.id)
	assert.Equal(t, 1, repo.ListForTenantCalls())
	assert.Equal(t, 2, repo.ByNamespaceCalls())

	union, _ := cache.ActionUnion()
	assert.Equal(t, sortedUnion([]string{prefixed(a.Namespace, "svc:a"), prefixed(b.Namespace, "svc:b")}), union)

	// Actions without a namespace never route.
	_, _, err = cache.Route(context.Background(), "svc:a")
	require.ErrorIs(t, err, errEndpointNotFound)
	assert.Equal(t, 2, repo.ByNamespaceCalls(), "an unparseable action does not look anything up")
}

// gatedEndpoints blocks GetByNamespace until released, to observe concurrent misses.
type gatedEndpoints struct {
	repository.ServerlessEndpointRepository
	gate    chan struct{}
	entered chan struct{}
}

func (g *gatedEndpoints) GetByNamespace(ctx context.Context, tenantId, ns uuid.UUID) (*sqlcv1.V1ServerlessEndpoint, error) {
	g.entered <- struct{}{}
	<-g.gate

	return g.ServerlessEndpointRepository.GetByNamespace(ctx, tenantId, ns)
}

// Concurrent misses on one namespace share one lookup.
func TestRoutingCacheCoalescesConcurrentMisses(t *testing.T) {
	repo := memrepo.New()
	tenant := uuid.New()
	l := zerolog.Nop()

	a := newEndpointRow(endpointSpec{tenantId: tenant, name: "a", enabled: true})
	a.RegisteredActions = []string{prefixed(a.Namespace, "svc:a")}

	gated := &gatedEndpoints{ServerlessEndpointRepository: repo.Endpoints(), gate: make(chan struct{}), entered: make(chan struct{}, 1)}
	cache := newRoutingCache(tenant, gated, fakeEnc{}, &l)
	require.NoError(t, cache.Load(context.Background()))

	repo.AddEndpoint(a)

	const routes = 16

	var wg sync.WaitGroup

	errs := make(chan error, routes)

	for i := 0; i < routes; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, _, err := cache.Route(context.Background(), prefixed(a.Namespace, "svc:a"))
			errs <- err
		}()
	}

	<-gated.entered
	time.Sleep(20 * time.Millisecond)
	close(gated.gate)
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	assert.Equal(t, 1, repo.ByNamespaceCalls(), "one lookup served every concurrent miss")
	assert.Equal(t, 1, repo.ListForTenantCalls())
}

// Reconcile drops deleted endpoints and fetches in full only the rows whose version the cache
// does not hold; an unchanged tenant transfers its ids and versions and nothing else.
func TestRoutingCacheReconcileFetchesChangedRowsOnly(t *testing.T) {
	repo := memrepo.New()
	tenant := uuid.New()
	l := zerolog.Nop()

	rows := make([]*sqlcv1.V1ServerlessEndpoint, 0, 3)

	for _, name := range []string{"a", "b", "c"} {
		row := newEndpointRow(endpointSpec{tenantId: tenant, name: name, enabled: true})
		row.RegisteredActions = []string{prefixed(row.Namespace, "svc:"+name)}
		repo.AddEndpoint(row)
		rows = append(rows, row)
	}

	cache := newRoutingCache(tenant, repo.Endpoints(), fakeEnc{}, &l)
	require.NoError(t, cache.Load(context.Background()))

	read := repo.ReadRows()
	require.NoError(t, cache.Reconcile(context.Background()))
	assert.Equal(t, read, repo.ReadRows(), "an unchanged tenant reads no row")
	assert.Equal(t, 1, repo.ListVersionsCalls())
	assert.Equal(t, 0, repo.ByIdsCalls())
	assert.Equal(t, uint64(1), cache.Revision())

	// One endpoint changes, one is deleted, one is new: only the changed and the new rows
	// are fetched, the deleted one is dropped, and the union follows.
	a, b, c := rows[0], rows[1], rows[2]
	repo.UpdateEndpoint(a.ID, func(ep *sqlcv1.V1ServerlessEndpoint) {
		ep.RegisteredActions = []string{prefixed(a.Namespace, "svc:a2")}
	})
	repo.RemoveEndpoint(b.ID)
	d := newEndpointRow(endpointSpec{tenantId: tenant, name: "d", enabled: true})
	d.RegisteredActions = []string{prefixed(d.Namespace, "svc:d")}
	repo.AddEndpoint(d)

	read = repo.ReadRows()
	require.NoError(t, cache.Reconcile(context.Background()))
	assert.Equal(t, read+2, repo.ReadRows(), "only the changed and the new rows are read")
	assert.Equal(t, 1, repo.ByIdsCalls())
	assert.Equal(t, 1, repo.ListForTenantCalls(), "reconcile never reloads the tenant")

	_, ok := cache.Endpoint(b.ID)
	assert.False(t, ok, "the deleted endpoint is dropped")

	union, _ := cache.ActionUnion()
	assert.Equal(t, sortedUnion([]string{prefixed(a.Namespace, "svc:a2"), prefixed(c.Namespace, "svc:c"), prefixed(d.Namespace, "svc:d")}), union)
}

func TestRoutingCacheRefreshAndUnion(t *testing.T) {
	repo := memrepo.New()
	tenant := uuid.New()
	l := zerolog.Nop()

	a := newEndpointRow(endpointSpec{tenantId: tenant, name: "a", enabled: true})
	a.RegisteredActions = []string{prefixed(a.Namespace, "svc:a")}
	repo.AddEndpoint(a)

	disabled := newEndpointRow(endpointSpec{tenantId: tenant, name: "off", enabled: false})
	disabled.RegisteredActions = []string{prefixed(disabled.Namespace, "svc:off")}
	repo.AddEndpoint(disabled)

	cache := newRoutingCache(tenant, repo.Endpoints(), fakeEnc{}, &l)
	require.NoError(t, cache.Load(context.Background()))

	union, rev := cache.ActionUnion()
	assert.Equal(t, []string{prefixed(a.Namespace, "svc:a")}, union, "disabled endpoints are excluded from the union")
	assert.Equal(t, uint64(1), rev, "the load published one revision")

	_, _, err := cache.Route(context.Background(), prefixed(disabled.Namespace, "svc:off"))
	assert.ErrorIs(t, err, errEndpointNotFound, "disabled endpoints do not route")

	// A registered_actions change written by the owner shows up through the incremental
	// refresh and changes the union.
	repo.UpdateEndpoint(a.ID, func(ep *sqlcv1.V1ServerlessEndpoint) {
		ep.RegisteredActions = []string{prefixed(a.Namespace, "svc:a"), prefixed(a.Namespace, "svc:a2")}
	})

	require.NoError(t, cache.Refresh(context.Background()))
	union, rev = cache.ActionUnion()
	assert.Equal(t, []string{prefixed(a.Namespace, "svc:a"), prefixed(a.Namespace, "svc:a2")}, union)
	assert.Equal(t, uint64(2), rev)
	assert.Equal(t, 1, repo.ListForTenantCalls(), "the initial load; the miss on the disabled endpoint looked it up by namespace")
	assert.Equal(t, 1, repo.ByNamespaceCalls())
	assert.Equal(t, 1, repo.ListSinceCalls())

	added, removed, current, ok := cache.DeltasSince(1)
	require.True(t, ok)
	assert.Equal(t, uint64(2), current, "the delta ends at the current revision")
	assert.Equal(t, []string{prefixed(a.Namespace, "svc:a2")}, added)
	assert.Empty(t, removed)

	// SetHealthcheck replaces the endpoint's actions in the union immediately.
	changed := cache.SetHealthcheck(a.ID, []string{prefixed(a.Namespace, "svc:new")})
	assert.True(t, changed)
	union, rev = cache.ActionUnion()
	assert.Equal(t, []string{prefixed(a.Namespace, "svc:new")}, union)
	assert.Equal(t, uint64(3), rev)
	assert.False(t, cache.SetHealthcheck(a.ID, []string{prefixed(a.Namespace, "svc:new")}), "same actions, no change")
	assert.Equal(t, uint64(3), cache.Revision(), "an unchanged union keeps its revision")

	// Deltas coalesce across revisions: svc:a2 entered at 2 and left at 3, so from 1 the net
	// change is svc:a out, svc:new in.
	added, removed, current, ok = cache.DeltasSince(1)
	require.True(t, ok)
	assert.Equal(t, uint64(3), current)
	assert.Equal(t, []string{prefixed(a.Namespace, "svc:new")}, added)
	assert.Equal(t, []string{prefixed(a.Namespace, "svc:a")}, removed)

	_, _, _, ok = cache.DeltasSince(7)
	assert.False(t, ok, "a revision the cache never published falls back to a full diff")

	// The fallback diffs a set against the union without the log.
	added, removed, current = cache.DiffAgainst(map[string]struct{}{prefixed(a.Namespace, "svc:a"): {}})
	assert.Equal(t, uint64(3), current)
	assert.Equal(t, []string{prefixed(a.Namespace, "svc:new")}, added)
	assert.Equal(t, []string{prefixed(a.Namespace, "svc:a")}, removed)

	// A refresh returning the row at the version the cache already applied changes nothing.
	require.NoError(t, cache.Refresh(context.Background()))
	assert.Equal(t, uint64(3), cache.Revision())

	// A hard-deleted endpoint survives incremental refreshes and is dropped by a full load.
	repo.RemoveEndpoint(a.ID)
	require.NoError(t, cache.Refresh(context.Background()))
	_, ok = cache.Endpoint(a.ID)
	assert.True(t, ok)

	require.NoError(t, cache.Load(context.Background()))
	_, ok = cache.Endpoint(a.ID)
	assert.False(t, ok)
	union, _ = cache.ActionUnion()
	assert.Empty(t, union)
}

// Two endpoints advertising the same action keep it in the union until the last one drops it.
func TestRoutingCacheUnionIsReferenceCounted(t *testing.T) {
	repo := memrepo.New()
	tenant := uuid.New()
	l := zerolog.Nop()

	shared := "shared:run"

	a := newEndpointRow(endpointSpec{tenantId: tenant, name: "a", enabled: true})
	a.RegisteredActions = []string{shared, prefixed(a.Namespace, "svc:a")}
	b := newEndpointRow(endpointSpec{tenantId: tenant, name: "b", enabled: true})
	b.RegisteredActions = []string{shared}
	repo.AddEndpoint(a)
	repo.AddEndpoint(b)

	cache := newRoutingCache(tenant, repo.Endpoints(), fakeEnc{}, &l)
	require.NoError(t, cache.Load(context.Background()))

	union, _ := cache.ActionUnion()
	assert.Equal(t, []string{prefixed(a.Namespace, "svc:a"), shared}, union)

	assert.False(t, cache.SetHealthcheck(b.ID, nil), "the action is still advertised by a")
	union, _ = cache.ActionUnion()
	assert.Contains(t, union, shared)

	assert.True(t, cache.SetHealthcheck(a.ID, []string{prefixed(a.Namespace, "svc:a")}), "the last advertiser dropped it")
	union, _ = cache.ActionUnion()
	assert.Equal(t, []string{prefixed(a.Namespace, "svc:a")}, union)

	// Disabling an endpoint removes its contribution through a refresh.
	repo.UpdateEndpoint(a.ID, func(ep *sqlcv1.V1ServerlessEndpoint) { ep.Enabled = false })
	require.NoError(t, cache.Refresh(context.Background()))
	union, _ = cache.ActionUnion()
	assert.Empty(t, union)
}
