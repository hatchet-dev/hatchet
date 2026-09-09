//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
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
	rows     []*sqlcv1.V1ServerlessEndpoint
	statuses []bool
	readRows int
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

	fake := &fakeSession{}
	union, rev := c.ActionUnion()
	reg := &registration{session: fake, advertised: union, advertisedRev: rev}
	l := zerolog.Nop()
	reg.r = &runner{l: &l}
	ts := &tenantState{cache: c, tenantId: budgetTenant, reg: reg}
	reg.ts = ts

	elapsed, allocated := measure(func() {
		for i := 0; i < 100; i++ {
			reg.r.syncTenantActions(context.Background(), ts)
		}
	})

	t.Logf("100 unchanged syncs: elapsed=%s allocated=%d", elapsed, allocated)

	assert.Equal(t, 0, fake.deltaCount())
	assert.Less(t, allocated, uint64(1<<20), "an unchanged union must not be copied and diffed on every sync")
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
