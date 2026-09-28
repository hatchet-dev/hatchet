//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
)

func TestCanonicalWorkflow(t *testing.T) {
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

	out, err := canonicalWorkflow(wf)
	require.NoError(t, err)

	assert.Equal(t, "echo", out.Name, "the workflow name is registered as declared")
	assert.Equal(t, []string{"user:created"}, out.EventTriggers, "event keys are registered as declared")
	assert.Equal(t, []string{"*/5 * * * *"}, out.CronTriggers)
	assert.Equal(t, "svc:run", out.Tasks[0].Action, "service and verb are normalized like the engine does")
	assert.Equal(t, "svc:other:sub", out.Tasks[1].Action)
	assert.Equal(t, "svc:fail", out.OnFailureTask.Action)

	// Deep copy: the input is untouched and canonicalizing twice is idempotent.
	assert.Equal(t, "Svc:Run", wf.Tasks[0].Action)

	again, err := canonicalWorkflow(out)
	require.NoError(t, err)
	assert.Equal(t, out.Tasks[0].Action, again.Tasks[0].Action)

	actions, err := actionsForWorkflow(out)
	require.NoError(t, err)
	assert.Equal(t, []string{"svc:run", "svc:other:sub", "svc:fail"}, actions)
}

func TestCanonicalWorkflowRejectsBadActions(t *testing.T) {
	_, err := canonicalWorkflow(&v1.CreateWorkflowVersionRequest{Name: "x", Tasks: []*v1.CreateTaskOpts{{Action: ""}}})
	assert.Error(t, err)

	_, err = canonicalWorkflow(&v1.CreateWorkflowVersionRequest{Name: "x", Tasks: []*v1.CreateTaskOpts{{Action: "noverb"}}})
	assert.Error(t, err)

	_, err = canonicalWorkflow(nil)
	assert.Error(t, err)
}

func TestNormalizeAction(t *testing.T) {
	got, err := normalizeAction("Svc:Run")
	require.NoError(t, err)
	assert.Equal(t, "svc:run", got)

	_, err = normalizeAction("svc:")
	assert.Error(t, err, "an empty verb is refused")

	_, err = normalizeAction(":run")
	assert.Error(t, err, "an empty service is refused")
}

// Route resolves an action to an enabled endpoint advertising it; an action none advertises
// fails at once, without any repository lookup.
func TestRoutingCacheRouteAndMiss(t *testing.T) {
	repo := memrepo.New()
	tenant := uuid.New()
	l := zerolog.Nop()

	a := newEndpointRow(endpointSpec{tenantId: tenant, name: "a", enabled: true, actions: []string{"svc:a"}})
	repo.AddEndpoint(a)

	cache := newRoutingCache(tenant, repo.Endpoints(), fakeEnc{}, &l)
	require.NoError(t, cache.Load(context.Background()))
	require.Equal(t, 1, repo.ListForTenantCalls())

	ep, cfg, err := cache.Route("svc:a")
	require.NoError(t, err)
	assert.Equal(t, a.ID, ep.id)
	assert.Equal(t, "secret-a", cfg.secret, "secret is decrypted once at load")
	assert.Equal(t, a.TriggerUrl, cfg.triggerUrl)
	assert.Equal(t, 1, repo.ListForTenantCalls(), "a hit does not reload")

	read := repo.ReadRows()

	_, _, err = cache.Route("svc:b")
	require.ErrorIs(t, err, errEndpointNotFound)
	assert.Equal(t, 1, repo.ListForTenantCalls(), "a miss does not reload the tenant")
	assert.Equal(t, 0, repo.ListSinceCalls(), "a miss does not refresh")
	assert.Equal(t, read, repo.ReadRows(), "a miss reads nothing")

	// The action routes once an endpoint advertising it reaches the cache.
	b := newEndpointRow(endpointSpec{tenantId: tenant, name: "b", enabled: true, actions: []string{"svc:b"}})
	repo.AddEndpoint(b)
	require.NoError(t, cache.Refresh(context.Background()))

	ep, _, err = cache.Route("svc:b")
	require.NoError(t, err)
	assert.Equal(t, b.ID, ep.id)

	union, _ := cache.ActionUnion()
	assert.Equal(t, []string{"svc:a", "svc:b"}, union)
}

// Two endpoints declaring the same action both serve it, like two workers: Route spreads
// deliveries over the ones known healthy, skips one marked unhealthy while another is not,
// falls back to any enabled one when none is known healthy, and never picks a disabled one.
func TestRoutingCacheSharedActionRoutesToEveryHealthyEndpoint(t *testing.T) {
	repo := memrepo.New()
	tenant := uuid.New()
	l := zerolog.Nop()

	a := newEndpointRow(endpointSpec{tenantId: tenant, name: "a", enabled: true, actions: []string{"svc:run"}})
	b := newEndpointRow(endpointSpec{tenantId: tenant, name: "b", enabled: true, actions: []string{"svc:run"}})
	off := newEndpointRow(endpointSpec{tenantId: tenant, name: "off", enabled: false, actions: []string{"svc:run"}})
	repo.AddEndpoint(a)
	repo.AddEndpoint(b)
	repo.AddEndpoint(off)

	cache := newRoutingCache(tenant, repo.Endpoints(), fakeEnc{}, &l)
	require.NoError(t, cache.Load(context.Background()))

	ids := []uuid.UUID{a.ID, b.ID}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	assert.Equal(t, ids, cache.Serving("svc:run"), "both enabled endpoints serve the action; the disabled one does not")

	routed := func(n int) map[uuid.UUID]int {
		seen := map[uuid.UUID]int{}

		for i := 0; i < n; i++ {
			ep, _, err := cache.Route("svc:run")
			require.NoError(t, err)
			seen[ep.id]++
		}

		return seen
	}

	seen := routed(200)
	assert.Contains(t, seen, a.ID, "a serves some deliveries")
	assert.Contains(t, seen, b.ID, "b serves some deliveries")
	assert.NotContains(t, seen, off.ID, "a disabled endpoint never routes")

	// An endpoint marked unhealthy is skipped while the other is healthy.
	cache.SetStatus(a.ID, false, "healthcheck returned status 503", time.Now())
	seen = routed(50)
	assert.Equal(t, map[uuid.UUID]int{b.ID: 50}, seen)

	// With every endpoint unhealthy the delivery still goes somewhere enabled.
	cache.SetStatus(b.ID, false, "healthcheck returned status 503", time.Now())
	seen = routed(200)
	assert.Contains(t, seen, a.ID)
	assert.Contains(t, seen, b.ID)
	assert.NotContains(t, seen, off.ID)

	// Recovery brings the healthy endpoint back as the only candidate.
	cache.SetStatus(a.ID, true, "", time.Now())
	seen = routed(50)
	assert.Equal(t, map[uuid.UUID]int{a.ID: 50}, seen)

	// The union names the shared action once; it stays while any enabled endpoint advertises
	// it and leaves when the last one stops.
	union, rev := cache.ActionUnion()
	assert.Equal(t, []string{"svc:run"}, union)

	assert.False(t, cache.SetHealthcheck(a.ID, nil, nil), "b still advertises the action")
	assert.Equal(t, rev, cache.Revision())
	assert.Equal(t, []uuid.UUID{b.ID}, cache.Serving("svc:run"))

	ep, _, err := cache.Route("svc:run")
	require.NoError(t, err)
	assert.Equal(t, b.ID, ep.id)

	assert.True(t, cache.SetHealthcheck(b.ID, nil, nil), "the last advertiser dropped it")
	added, removed, current, ok := cache.DeltasSince(rev)
	require.True(t, ok)
	assert.Equal(t, rev+1, current)
	assert.Empty(t, added)
	assert.Equal(t, []string{"svc:run"}, removed)
	assert.Empty(t, cache.Serving("svc:run"))

	_, _, err = cache.Route("svc:run")
	assert.ErrorIs(t, err, errEndpointNotFound)
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
		row.RegisteredActions = []string{"svc:" + name}
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
	a, b := rows[0], rows[1]
	repo.UpdateEndpoint(a.ID, func(ep *sqlcv1.V1ServerlessEndpoint) {
		ep.RegisteredActions = []string{"svc:a2"}
	})
	repo.RemoveEndpoint(b.ID)
	d := newEndpointRow(endpointSpec{tenantId: tenant, name: "d", enabled: true})
	d.RegisteredActions = []string{"svc:d"}
	repo.AddEndpoint(d)

	read = repo.ReadRows()
	require.NoError(t, cache.Reconcile(context.Background()))
	assert.Equal(t, read+2, repo.ReadRows(), "only the changed and the new rows are read")
	assert.Equal(t, 1, repo.ByIdsCalls())
	assert.Equal(t, 1, repo.ListForTenantCalls(), "reconcile never reloads the tenant")

	_, ok := cache.Endpoint(b.ID)
	assert.False(t, ok, "the deleted endpoint is dropped")

	union, _ := cache.ActionUnion()
	assert.Equal(t, sortedUnion([]string{"svc:a2", "svc:c", "svc:d"}), union)
}

func TestRoutingCacheRefreshAndUnion(t *testing.T) {
	repo := memrepo.New()
	tenant := uuid.New()
	l := zerolog.Nop()

	a := newEndpointRow(endpointSpec{tenantId: tenant, name: "a", enabled: true})
	a.RegisteredActions = []string{"svc:a"}
	repo.AddEndpoint(a)

	disabled := newEndpointRow(endpointSpec{tenantId: tenant, name: "off", enabled: false})
	disabled.RegisteredActions = []string{"svc:off"}
	repo.AddEndpoint(disabled)

	cache := newRoutingCache(tenant, repo.Endpoints(), fakeEnc{}, &l)
	require.NoError(t, cache.Load(context.Background()))

	union, rev := cache.ActionUnion()
	assert.Equal(t, []string{"svc:a"}, union, "disabled endpoints are excluded from the union")
	assert.Equal(t, uint64(1), rev, "the load published one revision")

	_, _, err := cache.Route("svc:off")
	assert.ErrorIs(t, err, errEndpointNotFound, "disabled endpoints do not route")

	// A registered_actions change written by the owner shows up through the incremental
	// refresh and changes the union.
	repo.UpdateEndpoint(a.ID, func(ep *sqlcv1.V1ServerlessEndpoint) {
		ep.RegisteredActions = []string{"svc:a", "svc:a2"}
	})

	require.NoError(t, cache.Refresh(context.Background()))
	union, rev = cache.ActionUnion()
	assert.Equal(t, []string{"svc:a", "svc:a2"}, union)
	assert.Equal(t, uint64(2), rev)
	assert.Equal(t, 1, repo.ListForTenantCalls(), "the initial load only")
	assert.Equal(t, 1, repo.ListSinceCalls())

	added, removed, current, ok := cache.DeltasSince(1)
	require.True(t, ok)
	assert.Equal(t, uint64(2), current, "the delta ends at the current revision")
	assert.Equal(t, []string{"svc:a2"}, added)
	assert.Empty(t, removed)

	// SetHealthcheck replaces the endpoint's actions in the union immediately.
	changed := cache.SetHealthcheck(a.ID, []string{"svc:new"}, nil)
	assert.True(t, changed)
	union, rev = cache.ActionUnion()
	assert.Equal(t, []string{"svc:new"}, union)
	assert.Equal(t, uint64(3), rev)
	assert.False(t, cache.SetHealthcheck(a.ID, []string{"svc:new"}, nil), "same actions, no change")
	assert.Equal(t, uint64(3), cache.Revision(), "an unchanged union keeps its revision")

	// Deltas coalesce across revisions: svc:a2 entered at 2 and left at 3, so from 1 the net
	// change is svc:a out, svc:new in.
	added, removed, current, ok = cache.DeltasSince(1)
	require.True(t, ok)
	assert.Equal(t, uint64(3), current)
	assert.Equal(t, []string{"svc:new"}, added)
	assert.Equal(t, []string{"svc:a"}, removed)

	_, _, _, ok = cache.DeltasSince(7)
	assert.False(t, ok, "a revision the cache never published falls back to a full diff")

	// The fallback diffs a set against the union without the log.
	added, removed, current = cache.DiffAgainst(map[string]struct{}{"svc:a": {}})
	assert.Equal(t, uint64(3), current)
	assert.Equal(t, []string{"svc:new"}, added)
	assert.Equal(t, []string{"svc:a"}, removed)

	// A refresh returning the row at the version the cache already applied changes nothing.
	require.NoError(t, cache.Refresh(context.Background()))
	assert.Equal(t, uint64(3), cache.Revision())

	// A deleted endpoint is versioned by its deletion: the incremental refresh drops it and
	// its actions leave the union.
	repo.RemoveEndpoint(a.ID)
	require.NoError(t, cache.Refresh(context.Background()))
	_, ok = cache.Endpoint(a.ID)
	assert.False(t, ok)
	assert.Equal(t, uint64(4), cache.Revision())
	union, _ = cache.ActionUnion()
	assert.Empty(t, union)

	require.NoError(t, cache.Load(context.Background()))
	_, ok = cache.Endpoint(a.ID)
	assert.False(t, ok)
}

// Two endpoints advertising the same action keep it in the union until the last one drops it.
func TestRoutingCacheUnionIsReferenceCounted(t *testing.T) {
	repo := memrepo.New()
	tenant := uuid.New()
	l := zerolog.Nop()

	shared := "shared:run"

	a := newEndpointRow(endpointSpec{tenantId: tenant, name: "a", enabled: true})
	a.RegisteredActions = []string{shared, "svc:a"}
	b := newEndpointRow(endpointSpec{tenantId: tenant, name: "b", enabled: true})
	b.RegisteredActions = []string{shared}
	repo.AddEndpoint(a)
	repo.AddEndpoint(b)

	cache := newRoutingCache(tenant, repo.Endpoints(), fakeEnc{}, &l)
	require.NoError(t, cache.Load(context.Background()))

	union, _ := cache.ActionUnion()
	assert.Equal(t, []string{shared, "svc:a"}, union)

	assert.False(t, cache.SetHealthcheck(b.ID, nil, nil), "the action is still advertised by a")
	union, _ = cache.ActionUnion()
	assert.Contains(t, union, shared)

	assert.True(t, cache.SetHealthcheck(a.ID, []string{"svc:a"}, nil), "the last advertiser dropped it")
	union, _ = cache.ActionUnion()
	assert.Equal(t, []string{"svc:a"}, union)

	// Disabling an endpoint removes its contribution through a refresh.
	repo.UpdateEndpoint(a.ID, func(ep *sqlcv1.V1ServerlessEndpoint) { ep.Enabled = false })
	require.NoError(t, cache.Refresh(context.Background()))
	union, _ = cache.ActionUnion()
	assert.Empty(t, union)
}

// A tenant larger than one page is loaded in pages of endpointPageSize, each published on
// its own, and an endpoint the listing no longer names is dropped once every page is in.
func TestRoutingCacheLoadPagesTheTenant(t *testing.T) {
	repo := memrepo.New()
	tenant := uuid.New()
	l := zerolog.Nop()

	const numEndpoints = int(endpointPageSize)*2 + 1

	for i := 0; i < numEndpoints; i++ {
		repo.AddEndpoint(newEndpointRow(endpointSpec{tenantId: tenant, name: fmt.Sprintf("ep-%d", i), enabled: true, actions: []string{fmt.Sprintf("svc:a%d", i)}}))
	}

	cache := newRoutingCache(tenant, repo.Endpoints(), fakeEnc{}, &l)
	require.NoError(t, cache.Load(context.Background()))

	assert.Equal(t, 3, repo.ListForTenantCalls(), "two full pages and a short last one")
	assert.Len(t, cache.Endpoints(), numEndpoints)

	union, _ := cache.ActionUnion()
	assert.Len(t, union, numEndpoints)

	gone := cache.Endpoints()[0]
	repo.RemoveEndpoint(gone.id)

	require.NoError(t, cache.Load(context.Background()))

	assert.Len(t, cache.Endpoints(), numEndpoints-1)
	_, ok := cache.Endpoint(gone.id)
	assert.False(t, ok, "an endpoint the listing no longer names is dropped")

	union, _ = cache.ActionUnion()
	assert.Len(t, union, numEndpoints-1)
}

// A deleted endpoint reaches the cache through the incremental refresh, versioned by its
// deletion, and is dropped there: its deliveries stop on the next refresh, before any
// reconcile, and its cached signing secret goes with it.
func TestRoutingCacheRefreshDropsDeletedEndpoints(t *testing.T) {
	repo := memrepo.New()
	tenant := uuid.New()
	l := zerolog.Nop()

	a := newEndpointRow(endpointSpec{tenantId: tenant, name: "a", enabled: true, actions: []string{"svc:a"}})
	b := newEndpointRow(endpointSpec{tenantId: tenant, name: "b", enabled: true, actions: []string{"svc:b"}})
	repo.AddEndpoint(a)
	repo.AddEndpoint(b)

	cache := newRoutingCache(tenant, repo.Endpoints(), fakeEnc{}, &l)
	require.NoError(t, cache.Load(context.Background()))
	require.NoError(t, cache.Refresh(context.Background()))

	_, _, err := cache.Route("svc:a")
	require.NoError(t, err)

	repo.RemoveEndpoint(a.ID)

	require.NoError(t, cache.Refresh(context.Background()))
	assert.Equal(t, 0, repo.ListVersionsCalls(), "no reconcile ran")

	_, ok := cache.Endpoint(a.ID)
	assert.False(t, ok, "the deleted endpoint left the cache on the refresh")
	assert.Empty(t, cache.Serving("svc:a"))

	_, _, err = cache.Route("svc:a")
	assert.ErrorIs(t, err, errEndpointNotFound, "nothing routes to a deleted endpoint")

	union, _ := cache.ActionUnion()
	assert.Equal(t, []string{"svc:b"}, union, "the deleted endpoint's actions left the union")

	// The refresh moved the watermark past the deletion: a further refresh reads nothing.
	read := repo.ReadRows()
	require.NoError(t, cache.Refresh(context.Background()))
	assert.Equal(t, read, repo.ReadRows())

	// Purging the row changes nothing the cache holds.
	n, err := repo.Endpoints().PurgeDeleted(context.Background(), time.Now().Add(time.Second))
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	require.NoError(t, cache.Reconcile(context.Background()))
	assert.Len(t, cache.Endpoints(), 1)
}
