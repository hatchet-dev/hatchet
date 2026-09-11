//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/lease"
)

// The tests in this file hold the routing cache to the budgets the performance review asked
// for: loading a tenant is linear in its actions, an unchanged refresh reads nothing, an
// unchanged union costs a registration nothing, and a status transition is one write.

// budgetEnc returns the first 32 bytes of the ciphertext, a cheap stand-in for decryption.
type budgetEnc struct{ encryption.EncryptionService }

func (budgetEnc) DecryptString(s, _ string) (string, error) { return strings.Clone(s[:32]), nil }

// budgetRepo is a lean endpoint repository over a fixed row set that counts what it returns.
type budgetRepo struct {
	repository.ServerlessEndpointRepository
	rows         []*sqlcv1.V1ServerlessEndpoint
	versions     []repository.ServerlessEndpointVersion
	statuses     []bool
	readRows     int
	readVersions int
}

func (r *budgetRepo) ListForTenant(context.Context, uuid.UUID) ([]*sqlcv1.V1ServerlessEndpoint, error) {
	r.readRows += len(r.rows)
	return r.rows, nil
}

func (r *budgetRepo) ListUpdatedSince(_ context.Context, _ uuid.UUID, since time.Time, sinceId uuid.UUID) ([]*sqlcv1.V1ServerlessEndpoint, error) {
	out := make([]*sqlcv1.V1ServerlessEndpoint, 0)

	for _, row := range r.rows {
		if row.UpdatedAt.Time.After(since) || (row.UpdatedAt.Time.Equal(since) && row.ID.String() > sinceId.String()) {
			out = append(out, row)
		}
	}

	r.readRows += len(out)

	return out, nil
}

// ListVersions pages a version list built once, so what the measurement sees is the
// cache's own work, not the fake's.
func (r *budgetRepo) ListVersions(_ context.Context, _ uuid.UUID, after repository.ServerlessEndpointVersion, limit int64) ([]repository.ServerlessEndpointVersion, error) {
	if r.versions == nil {
		r.versions = make([]repository.ServerlessEndpointVersion, 0, len(r.rows))

		for _, row := range r.rows {
			r.versions = append(r.versions, repository.ServerlessEndpointVersion{ID: row.ID, Version: row.UpdatedAt.Time})
		}

		sort.Slice(r.versions, func(i, j int) bool {
			if !r.versions[i].Version.Equal(r.versions[j].Version) {
				return r.versions[i].Version.Before(r.versions[j].Version)
			}

			return r.versions[i].ID.String() < r.versions[j].ID.String()
		})
	}

	start := 0

	for start < len(r.versions) {
		v := r.versions[start]

		if v.Version.After(after.Version) || (v.Version.Equal(after.Version) && bytes.Compare(v.ID[:], after.ID[:]) > 0) {
			break
		}

		start++
	}

	end := min(start+int(limit), len(r.versions))
	r.readVersions += end - start

	return r.versions[start:end], nil
}

func (r *budgetRepo) ListByIds(_ context.Context, ids []uuid.UUID) ([]*sqlcv1.V1ServerlessEndpoint, error) {
	want := make(map[uuid.UUID]struct{}, len(ids))

	for _, id := range ids {
		want[id] = struct{}{}
	}

	out := make([]*sqlcv1.V1ServerlessEndpoint, 0, len(ids))

	for _, row := range r.rows {
		if _, ok := want[row.ID]; ok {
			out = append(out, row)
		}
	}

	r.readRows += len(out)

	return out, nil
}

func (r *budgetRepo) ListForUnits(_ context.Context, _ []repository.ServerlessUnit, after uuid.UUID, limit int64) ([]*sqlcv1.V1ServerlessEndpoint, error) {
	out := make([]*sqlcv1.V1ServerlessEndpoint, 0)

	for _, row := range r.rows {
		if row.ID.String() > after.String() {
			out = append(out, row)

			if int64(len(out)) == limit {
				break
			}
		}
	}

	return out, nil
}

func (r *budgetRepo) UpdateStatus(_ context.Context, _ uuid.UUID, healthy bool, _ *string) (time.Time, error) {
	r.statuses = append(r.statuses, healthy)
	return time.Now(), nil
}

type budgetRootRepo struct {
	repository.ServerlessRepository
	ep *budgetRepo
}

func (r budgetRootRepo) Endpoints() repository.ServerlessEndpointRepository { return r.ep }

var budgetTenant = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// budgetRows builds n endpoints of one tenant with a actions each, all on shard 0, with ids
// ordered so keyset paging visits them in order.
func budgetRows(n, a int) []*sqlcv1.V1ServerlessEndpoint {
	rows := make([]*sqlcv1.V1ServerlessEndpoint, n)
	stamp := time.Now().Add(-time.Hour)

	for i := range rows {
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0001-%012d", i))
		actions := make([]string, a)

		for j := range actions {
			actions[j] = id.String() + fmt.Sprintf("_service:action%03d", j)
		}

		rows[i] = &sqlcv1.V1ServerlessEndpoint{
			ID:                id,
			TenantID:          budgetTenant,
			Namespace:         id,
			Name:              fmt.Sprintf("endpoint-%08d", i),
			HealthcheckUrl:    fmt.Sprintf("https://endpoint-%08d.example.test/health", i),
			TriggerUrl:        fmt.Sprintf("https://endpoint-%08d.example.test/trigger", i),
			SigningSecretEnc:  strings.Repeat("s", 100),
			RegisteredActions: actions,
			Enabled:           true,
			Healthy:           pgtype.Bool{Bool: true, Valid: true},
			UpdatedAt:         pgtype.Timestamptz{Time: stamp, Valid: true},
		}
	}

	return rows
}

func budgetCache(t *testing.T, n, a int) (*routingCache, *budgetRepo) {
	t.Helper()

	l := zerolog.Nop()
	repo := &budgetRepo{rows: budgetRows(n, a)}
	c := newRoutingCache(budgetTenant, repo, budgetEnc{}, &l)
	require.NoError(t, c.Load(context.Background()))

	return c, repo
}

// measure runs fn once and returns the wall time and bytes allocated.
func measure(fn func()) (time.Duration, uint64) {
	runtime.GC()

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)

	return elapsed, after.TotalAlloc - before.TotalAlloc
}

// Gaining a unit of 3,000 endpoints with 10 actions each must cost time and allocation linear
// in the actions: the review measured 14 s and 18 GB when the union was rebuilt per row.
func TestLoadUnitEndpointsBudget(t *testing.T) {
	const (
		endpoints  = 3000
		maxElapsed = 200 * time.Millisecond
		maxAlloc   = 50 << 20
	)

	l := zerolog.Nop()
	repo := &budgetRepo{rows: budgetRows(endpoints, 10)}
	r := &runner{repo: budgetRootRepo{ep: repo}, enc: budgetEnc{}, l: &l, tenants: map[uuid.UUID]*tenantState{}}
	units := []lease.Unit{{TenantId: budgetTenant}}
	ctx := context.Background()

	// Gaining the tenant's first unit loads the tenant in full; gaining another unit later
	// pages that unit's endpoints over the loaded cache. Both are measured.
	elapsed, allocated := measure(func() {
		ts := r.tenantFor(budgetTenant)
		require.NoError(t, r.loadTenant(ctx, ts, []int32{0}))
		require.NoError(t, r.loadUnitEndpoints(ctx, ts, units))
	})

	t.Logf("endpoints=%d elapsed=%s allocated=%d", endpoints, elapsed, allocated)

	assert.Less(t, elapsed, maxElapsed, "loading a unit must not rebuild the union per row")
	assert.Less(t, allocated, uint64(maxAlloc), "loading a unit must not allocate the union per row")

	ts := r.tenants[budgetTenant]
	require.NotNil(t, ts)
	union, rev := ts.cache.ActionUnion()
	assert.Len(t, union, endpoints*10)
	assert.Equal(t, uint64(1), rev, "the load published one revision and the unit pages none")
}

// A refresh with nothing changed must read nothing and leave the watermark alone: the review
// observed every refresh rereading the rows sharing the watermark's timestamp.
func TestRefreshWithoutChangesReadsNothing(t *testing.T) {
	c, repo := budgetCache(t, 1000, 10)
	since := c.since

	for i := 0; i < 5; i++ {
		repo.readRows = 0
		require.NoError(t, c.Refresh(context.Background()))
		assert.Equal(t, 0, repo.readRows, "refresh %d reread rows without a change", i+1)
		assert.Equal(t, since, c.since, "the watermark must not move without a change")
	}
}

// One endpoint's healthcheck change against a large tenant must cost that endpoint's actions,
// not a rebuild of the tenant's union.
func TestHealthcheckChangeCostIsLocal(t *testing.T) {
	c, repo := budgetCache(t, 10000, 10)
	ep := repo.rows[0]
	next := append([]string{}, ep.RegisteredActions[1:]...)
	next = append(next, ep.ID.String()+"_service:added")

	elapsed, allocated := measure(func() {
		for i := 0; i < 100; i++ {
			c.SetHealthcheck(ep.ID, next)
			c.SetHealthcheck(ep.ID, ep.RegisteredActions)
		}
	})

	t.Logf("200 changes: elapsed=%s allocated=%d", elapsed, allocated)

	assert.Less(t, elapsed, 200*time.Millisecond)
	assert.Less(t, allocated, uint64(20<<20))
}

// An unchanged union must cost a registration's sync nothing beyond a revision comparison.
func TestUnchangedUnionSyncIsFree(t *testing.T) {
	c, _ := budgetCache(t, 10000, 10)

	fake := newFakeSession(nil, operator.Registration{})
	reg := newRegistrationForTest(c, fake)
	ts := reg.ts

	elapsed, allocated := measure(func() {
		for i := 0; i < 100; i++ {
			reg.r.syncTenantActions(context.Background(), ts)
		}
	})

	t.Logf("100 unchanged syncs: elapsed=%s allocated=%d", elapsed, allocated)

	assert.Equal(t, 0, fake.deltaCount())
	assert.Less(t, allocated, uint64(1<<20), "an unchanged union must not be copied and diffed on every sync")
}

// A changed union must cost a registration's sync the delta, never the tenant's union: a
// hundred one-action changes on a tenant of 100k actions must not copy or sort the union a
// hundred times.
func TestChangedUnionSyncIsIncremental(t *testing.T) {
	c, repo := budgetCache(t, 10000, 10)
	ep := repo.rows[0]

	fake := newFakeSession(nil, operator.Registration{})
	reg := newRegistrationForTest(c, fake)
	next := append([]string{}, ep.RegisteredActions...)
	next = append(next, ep.ID.String()+"_service:added")

	elapsed, allocated := measure(func() {
		for i := 0; i < 50; i++ {
			c.SetHealthcheck(ep.ID, next)
			require.NoError(t, reg.syncActions(context.Background(), c))
			c.SetHealthcheck(ep.ID, ep.RegisteredActions)
			require.NoError(t, reg.syncActions(context.Background(), c))
		}
	})

	t.Logf("100 one-action changes and syncs over 100k actions: elapsed=%s allocated=%d", elapsed, allocated)

	assert.Equal(t, 100, fake.deltaCount())
	assert.Equal(t, 100, fake.flushCount())
	assert.Less(t, allocated, uint64(4<<20), "a one-action change must not allocate the union")
	assert.Less(t, elapsed, 500*time.Millisecond)
}

// A registration further behind than the cache's log falls back to a diff of the set it
// advertised against the union, and the deltas it applied since it opened are part of that
// set; after many syncs the applied deltas are folded into a new base. The engine's set must
// equal the union throughout.
func TestSyncFallsBackToDiffAndCompacts(t *testing.T) {
	c, repo := budgetCache(t, 4, 2)
	fake := newFakeSession(nil, operator.Registration{})
	reg := newRegistrationForTest(c, fake)

	engine := map[string]struct{}{}

	for _, id := range reg.base {
		engine[id] = struct{}{}
	}

	sync := func() {
		before := fake.deltaCount()
		require.NoError(t, reg.syncActions(context.Background(), c))

		for _, d := range fake.deltas[before:] {
			applyDelta(engine, d.add, d.remove)
		}
	}

	ep := repo.rows[0]

	// Sync after every change: the log covers each step.
	for i := 0; i < 3; i++ {
		c.SetHealthcheck(ep.ID, []string{ep.ID.String() + fmt.Sprintf("_service:v%d", i)})
		sync()
	}

	// More changes than the log keeps, then one sync: the fallback diff.
	for i := 0; i < unionLogSize+8; i++ {
		c.SetHealthcheck(ep.ID, []string{ep.ID.String() + fmt.Sprintf("_service:w%d", i)})
	}

	sync()

	union, rev := c.ActionUnion()
	assert.Equal(t, rev, reg.advertisedRev)
	assert.Equal(t, union, sortedUnion(keys(engine)), "the engine holds the union after the fallback")
	assert.Empty(t, reg.applied, "the fallback rebases the registration")

	// Enough small deltas to cross the compaction floor fold into a new base.
	for i := 0; i < appliedCompactionFloor+16; i++ {
		c.SetHealthcheck(ep.ID, []string{ep.ID.String() + fmt.Sprintf("_service:x%d", i)})
		sync()
	}

	union, rev = c.ActionUnion()
	assert.Equal(t, rev, reg.advertisedRev)
	assert.Equal(t, union, sortedUnion(keys(engine)))
	assert.Less(t, reg.appliedIds, len(reg.base)+appliedCompactionFloor+1)
	assert.Equal(t, union, sortedUnion(keys(reg.advertisedSet())), "the registration's own view matches the union")
}

func keys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))

	for id := range set {
		out = append(out, id)
	}

	return out
}

// The periodic anti-entropy pass of an unchanged tenant must transfer ids and versions only:
// no row, and no per-endpoint configuration built just to find it unchanged.
func TestUnchangedReconcileReadsNoRows(t *testing.T) {
	c, repo := budgetCache(t, 10000, 10)

	// One pass first, so the fake's version list exists and the measurement is the cache's.
	require.NoError(t, c.Reconcile(context.Background()))
	repo.readRows = 0
	repo.readVersions = 0

	elapsed, allocated := measure(func() {
		require.NoError(t, c.Reconcile(context.Background()))
	})

	t.Logf("unchanged reconcile of 10k endpoints: elapsed=%s allocated=%d versions=%d rows=%d", elapsed, allocated, repo.readVersions, repo.readRows)

	assert.Equal(t, 0, repo.readRows, "an unchanged tenant reads no row")
	assert.Equal(t, 10000, repo.readVersions)
	assert.Less(t, allocated, uint64(1<<20), "no row, configuration or per-endpoint bookkeeping is allocated")
	assert.Less(t, elapsed, 200*time.Millisecond)
}

// A full reload must observe a status another owner wrote, even though status writes leave
// updated_at alone.
func TestFullReloadObservesExternalStatusChange(t *testing.T) {
	env := newTestEnv(t)
	row := healthyRow(endpointSpec{tenantId: uuid.New(), name: "health-refresh", actions: []string{"svc:run"}})
	env.addEndpoint(row)

	cache := newRoutingCache(row.TenantID, env.repo.Endpoints(), fakeEnc{}, env.r.l)
	ctx := context.Background()
	require.NoError(t, cache.Load(ctx))

	_, err := env.repo.Endpoints().UpdateStatus(ctx, row.ID, false, nil)
	require.NoError(t, err)
	require.NoError(t, cache.Load(ctx))

	ep, _ := cache.Endpoint(row.ID)
	assert.False(t, cache.Config(ep).healthy, "a full reload must take the status another owner wrote")

	// The incremental refresh sees it too.
	_, err = env.repo.Endpoints().UpdateStatus(ctx, row.ID, true, nil)
	require.NoError(t, err)
	require.NoError(t, cache.Refresh(ctx))
	assert.True(t, cache.Config(ep).healthy, "a refresh must surface a status transition")
}

// A process that already caches a tenant, then gains one of its units, must recover an
// endpoint the previous owner left unhealthy: its stale cached health must not suppress the
// recovery write.
func TestRecoveryWriteAfterOwnershipTransfer(t *testing.T) {
	env := newTestEnv(t)
	row := healthyRow(endpointSpec{tenantId: uuid.New(), name: "health-recovery", actions: []string{"svc:run"}})
	env.addEndpoint(row)

	ctx := context.Background()
	ts := env.r.tenantFor(row.TenantID)
	require.NoError(t, ts.cache.Load(ctx))

	// The previous owner marks the endpoint unhealthy; this process refreshes and then takes
	// the unit over.
	_, err := env.repo.Endpoints().UpdateStatus(ctx, row.ID, false, nil)
	require.NoError(t, err)
	require.NoError(t, ts.cache.Load(ctx))

	ep, _ := ts.cache.Endpoint(row.ID)
	p := newEndpointPoller(env.r, ts, ep)

	for i := 0; i < 3; i++ {
		p.pollOnce(ctx)
	}

	db := env.repo.Endpoint(row.ID)
	require.NotNil(t, db)
	assert.True(t, db.Healthy.Bool, "successful polls after a takeover must write the recovery")
	assert.Len(t, env.repo.StatusWrites(), 2, "one unhealthy write by the previous owner, one recovery write")
}

// rejectingSession fails every PutWorkflow, as an engine that rejects the workflow does.
type rejectingSession struct{ operator.Session }

func (rejectingSession) PutWorkflow(context.Context, *v1.CreateWorkflowVersionRequest) ([]string, error) {
	return nil, fmt.Errorf("persistent workflow rejection")
}

type validHealthSender struct{}

func (validHealthSender) Deliver(context.Context, string, string, []byte, http.Header) (*safeclient.DeliveryResult, error) {
	return &safeclient.DeliveryResult{StatusCode: 200, BodyPrefix: []byte(`{"workflows":[{"name":"w","tasks":[{"readableId":"task","action":"svc:run"}]}]}`)}, nil
}

// A healthcheck that succeeds over HTTP but whose workflows the engine keeps rejecting is one
// status write (unhealthy with the engine's error), not a healthy/unhealthy pair per poll.
func TestPersistentWorkflowRejectionWritesStatusOnce(t *testing.T) {
	c, repo := budgetCache(t, 1, 0)
	ep := c.byId[repo.rows[0].ID]

	ts := &tenantState{cache: c, reg: &registration{session: rejectingSession{}}, hcSem: make(chan struct{}, 4)}
	l := zerolog.Nop()
	r := &runner{repo: budgetRootRepo{ep: repo}, cfg: DefaultConfig(), sender: validHealthSender{}, l: &l, m: newMetrics("budget-status"), hcSem: make(chan struct{}, 256)}
	ts.reg.r = r
	p := newEndpointPoller(r, ts, ep)

	for i := 0; i < 3; i++ {
		p.pollOnce(context.Background())
	}

	t.Logf("status writes after three identical polls: %v", repo.statuses)

	assert.Equal(t, []bool{false}, repo.statuses, "the rejection is written once, as the poll's single outcome")
	assert.True(t, ep.cfg.healthKnown)
	assert.False(t, ep.cfg.healthy)
	assert.Contains(t, ep.cfg.statusError, "persistent workflow rejection")
}

// newRegistrationForTest builds a registration advertising the cache's current union over
// session, the way openRegistration does, with a quiet runner.
func newRegistrationForTest(c *routingCache, session operator.Session) *registration {
	l := zerolog.Nop()
	union, rev := c.ActionUnion()
	reg := &registration{session: session, base: union, advertisedRev: rev, r: &runner{l: &l}}
	reg.ts = &tenantState{cache: c, tenantId: c.tenantId, reg: reg}

	return reg
}

// noopSession accepts every delta without recording it, for measuring the sync itself.
type noopSession struct{ operator.Session }

func (noopSession) AddActions(context.Context, []string) error    { return nil }
func (noopSession) RemoveActions(context.Context, []string) error { return nil }
func (noopSession) Flush(context.Context) error                   { return nil }

// BenchmarkOneActionChangeSync is one endpoint's healthcheck adding an action to a tenant of
// n endpoints with ten actions each, followed by the owner's sync, the way applyChange runs
// them. The cost must be that of the delta, not of the tenant's union.
func BenchmarkOneActionChangeSync(b *testing.B) {
	for _, n := range []int{1000, 10000, 100000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			l := zerolog.Nop()
			repo := &budgetRepo{rows: budgetRows(n, 10)}
			c := newRoutingCache(budgetTenant, repo, budgetEnc{}, &l)

			if err := c.Load(context.Background()); err != nil {
				b.Fatal(err)
			}

			ep := repo.rows[0]
			reg := newRegistrationForTest(c, noopSession{})
			base := append([]string(nil), ep.RegisteredActions...)
			variants := [][]string{
				append(append([]string(nil), base...), ep.Namespace.String()+"_service:extra0"),
				append(append([]string(nil), base...), ep.Namespace.String()+"_service:extra1"),
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				c.SetHealthcheck(ep.ID, variants[i%2])

				if err := reg.syncActions(context.Background(), c); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
