//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// workerActionsPool returns a migrated database for the worker action tests. It connects to
// HATCHET_TEST_DATABASE_URL when set, so a local run can reuse a migrated database instead of
// starting a container; otherwise it starts one through setupPostgresWithMigration.
func workerActionsPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if dsn := os.Getenv("HATCHET_TEST_DATABASE_URL"); dsn != "" {
		pool, err := pgxpool.New(context.Background(), dsn)
		require.NoError(t, err)
		t.Cleanup(pool.Close)

		return pool
	}

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	return pool
}

func seedDispatcher(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()

	dispatcherId := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO "Dispatcher" ("id", "lastHeartbeatAt") VALUES ($1, now())`, dispatcherId)
	require.NoError(t, err)

	return dispatcherId
}

// xorFold is the XOR combination of per-action digests: the combination the hash must not
// be, because sums of digests can be solved for a chosen value.
func xorFold(actionIds []string) []byte {
	out := hashActions(nil)

	for _, actionId := range actionIds {
		digest := sha256.Sum256([]byte(actionId))

		for i := range out {
			out[i] ^= digest[i]
		}
	}

	return out
}

// The hash is a function of the final set alone: construction order, casing, duplicates and
// empty entries do not change it, and it is not the XOR of per-action digests.
func TestHashActionsIsCanonical(t *testing.T) {
	base := hashActions([]string{"svc:a", "svc:b"})

	assert.Equal(t, base, hashActions([]string{"svc:b", "svc:a"}), "order")
	assert.Equal(t, base, hashActions([]string{"SVC:A", "svc:B"}), "casing")
	assert.Equal(t, base, hashActions([]string{"svc:a", "svc:b", "svc:a", ""}), "duplicates and empties")
	assert.NotEqual(t, base, hashActions([]string{"svc:a"}))
	assert.NotEqual(t, base, hashActions([]string{"svc:a", "svc:b", "svc:c"}))
	assert.NotEqual(t, hashActions(nil), hashActions([]string{"svc:a"}))
	assert.NotEqual(t, base, xorFold([]string{"svc:a", "svc:b"}), "the hash is not a linear combination of per-action digests")

	// the separator keeps ids whose concatenation is the same apart
	assert.NotEqual(t, hashActions([]string{"svc:ab", "svc:c"}), hashActions([]string{"svc:a", "svc:bc"}), "equal concatenation")

	// the framing is main's, so a hash written by an older engine equals the one this engine
	// computes for the same set
	pinned := sha256.Sum256([]byte("svc:a;svc:b;"))
	assert.Equal(t, pinned[:], base, "each sorted id followed by a semicolon")
}

// A worker created with an initial action set and a worker built by deltas from an empty set
// hash equal for the same final set, and both equal the canonical digest.
func TestWorkerActionHashAgreesAcrossPaths(t *testing.T) {
	pool := workerActionsPool(t)
	repo := createWorkerActionsRepositoryForTest(pool)
	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)
	dispatcherId := seedDispatcher(t, ctx, pool)
	operatorId := uuid.New()

	// an operator worker of the unmetered kind: this repository has no limit meter, and the
	// hash is what is under test
	created, err := repo.CreateNewWorker(ctx, tenantId, &CreateWorkerOpts{
		DispatcherId: dispatcherId,
		Name:         "initial-set",
		Actions:      []string{"Svc:Run", "svc:other"},
		OperatorId:   &operatorId,
		OperatorKind: sqlcv1.V1OperatorKindDAG,
	})
	require.NoError(t, err)

	incremental := seedWorkerForActions(t, ctx, pool, tenantId)
	_, err = addWorkerActions(repo, ctx, tenantId, incremental, []string{"svc:other"})
	require.NoError(t, err)
	_, err = addWorkerActions(repo, ctx, tenantId, incremental, []string{"svc:run"})
	require.NoError(t, err)

	churned := seedWorkerForActions(t, ctx, pool, tenantId)
	_, err = addWorkerActions(repo, ctx, tenantId, churned, []string{"svc:run", "svc:extra", "svc:other"})
	require.NoError(t, err)
	_, err = removeWorkerActions(repo, ctx, tenantId, churned, []string{"svc:extra"})
	require.NoError(t, err)

	want := hashActions([]string{"svc:run", "svc:other"})

	assert.Equal(t, want, workerActionHash(t, ctx, pool, created.ID), "initial set")
	assert.Equal(t, want, refreshedHash(t, ctx, repo, pool, tenantId, incremental), "deltas")
	assert.Equal(t, want, refreshedHash(t, ctx, repo, pool, tenantId, churned), "add then remove")
	assert.Equal(t, []string{"svc:other", "svc:run"}, linkedActions(t, ctx, pool, created.ID))
	assert.Equal(t, 2, workerActionCount(t, ctx, pool, created.ID), "the initial set is counted")
	assert.Equal(t, 2, workerActionCount(t, ctx, pool, churned))
}

// Action links are only ever made between a worker and actions of the worker's own tenant:
// a caller that pairs a tenant with another tenant's worker is refused before any mutation.
func TestWorkerActionsRejectTenantWorkerMismatch(t *testing.T) {
	pool := workerActionsPool(t)
	repo := createWorkerActionsRepositoryForTest(pool)
	ctx := context.Background()
	tenantA := seedTenantForActions(t, ctx, pool)
	tenantB := seedTenantForActions(t, ctx, pool)
	worker := seedWorkerForActions(t, ctx, pool, tenantB)

	_, err := addWorkerActions(repo, ctx, tenantA, worker, []string{"svc:run"})
	require.Error(t, err, "a worker of another tenant must not be linked")
	assert.Empty(t, linkedActions(t, ctx, pool, worker))

	_, err = addWorkerActions(repo, ctx, tenantB, worker, []string{"svc:run"})
	require.NoError(t, err)

	_, err = removeWorkerActions(repo, ctx, tenantA, worker, []string{"svc:run"})
	require.Error(t, err, "a worker of another tenant must not be unlinked")
	assert.Equal(t, []string{"svc:run"}, linkedActions(t, ctx, pool, worker))

	var cross int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM "_ActionToWorker" aw JOIN "Action" a ON a.id = aw."A" JOIN "Worker" w ON w.id = aw."B" WHERE aw."B" = $1 AND a."tenantId" <> w."tenantId"`,
		worker,
	).Scan(&cross))
	assert.Zero(t, cross)
}

// A budgeted add links nothing when it would leave the operator over its cap, repeated actions
// never count against it, and the hash of a refused delta's worker is untouched.
func TestAddWorkerActionsWithinBudget(t *testing.T) {
	pool := workerActionsPool(t)
	repo := createWorkerActionsRepositoryForTest(pool)
	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)
	operatorId := seedOperator(t, ctx, pool, tenantId)
	worker := seedOperatorWorker(t, ctx, pool, tenantId, operatorId)
	initial := workerActionHash(t, ctx, pool, worker)

	added, err := addWorkerActionsWithinBudget(repo, ctx, tenantId, worker, []string{"svc:a", "svc:b"}, 1)
	require.ErrorIs(t, err, ErrWorkerActionBudgetExceeded)
	assert.Zero(t, added)
	assert.Empty(t, linkedActions(t, ctx, pool, worker), "a refused delta links nothing")
	assert.Equal(t, initial, workerActionHash(t, ctx, pool, worker))

	added, err = addWorkerActionsWithinBudget(repo, ctx, tenantId, worker, []string{"svc:a", "svc:b"}, 2)
	require.NoError(t, err)
	assert.Equal(t, 2, added)

	added, err = addWorkerActionsWithinBudget(repo, ctx, tenantId, worker, []string{"svc:a", "svc:b"}, 2)
	require.NoError(t, err, "repeated actions do not consume budget")
	assert.Zero(t, added)

	added, err = addWorkerActionsWithinBudget(repo, ctx, tenantId, worker, []string{"svc:b", "svc:c"}, 2)
	require.ErrorIs(t, err, ErrWorkerActionBudgetExceeded)
	assert.Zero(t, added)
	assert.Equal(t, []string{"svc:a", "svc:b"}, linkedActions(t, ctx, pool, worker))

	added, err = addWorkerActionsWithinBudget(repo, ctx, tenantId, worker, []string{"svc:b", "svc:c"}, -1)
	require.NoError(t, err, "a negative budget is unlimited")
	assert.Equal(t, 1, added)
}

func actionRowVersions(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantId uuid.UUID) []string {
	t.Helper()

	rows, err := pool.Query(ctx, `SELECT xmin::text FROM "Action" WHERE "tenantId" = $1 ORDER BY "actionId"`, tenantId)
	require.NoError(t, err)
	defer rows.Close()

	var versions []string

	for rows.Next() {
		var v string
		require.NoError(t, rows.Scan(&v))
		versions = append(versions, v)
	}

	require.NoError(t, rows.Err())

	return versions
}

// Workers replaying the same existing actions in different orders must neither deadlock on
// the shared Action rows nor rewrite them.
func TestWorkerActionsConcurrentReplayOfExistingActions(t *testing.T) {
	pool := workerActionsPool(t)
	repo := createWorkerActionsRepositoryForTest(pool)
	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)

	const total = 1000

	ids := make([]string, 0, total)

	for i := 0; i < total; i++ {
		ids = append(ids, fmt.Sprintf("replay:action%04d", i))
	}

	seed := seedWorkerForActions(t, ctx, pool, tenantId)
	_, err := addWorkerActions(repo, ctx, tenantId, seed, ids)
	require.NoError(t, err)

	before := actionRowVersions(t, ctx, pool, tenantId)
	require.Len(t, before, total)

	const workers = 4

	workerIds := make([]uuid.UUID, 0, workers)

	for i := 0; i < workers; i++ {
		workerIds = append(workerIds, seedWorkerForActions(t, ctx, pool, tenantId))
	}

	var wg sync.WaitGroup
	errs := make(chan error, workers)

	for i, workerId := range workerIds {
		shuffled := append([]string(nil), ids...)
		rand.New(rand.NewSource(int64(i+1))).Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] }) // nolint: gosec

		wg.Add(1)

		go func(workerId uuid.UUID, ids []string) {
			defer wg.Done()

			added, err := addWorkerActions(repo, ctx, tenantId, workerId, ids)

			if err == nil && added != total {
				err = fmt.Errorf("linked %d actions, want %d", added, total)
			}

			errs <- err
		}(workerId, shuffled)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	assert.Equal(t, before, actionRowVersions(t, ctx, pool, tenantId), "existing Action rows are not rewritten by a replay")

	want := workerActionHash(t, ctx, pool, seed)

	for _, workerId := range workerIds {
		assert.Equal(t, want, workerActionHash(t, ctx, pool, workerId))
		assert.Len(t, linkedActions(t, ctx, pool, workerId), total)
	}
}

// TestWorkerActionDeltaCostAtScale measures one delta against a worker holding 100,000
// actions. It only runs when HATCHET_WORKER_ACTIONS_SCALE is set.
func TestWorkerActionDeltaCostAtScale(t *testing.T) {
	if os.Getenv("HATCHET_WORKER_ACTIONS_SCALE") == "" {
		t.Skip("set HATCHET_WORKER_ACTIONS_SCALE to run the scale measurement")
	}

	pool := workerActionsPool(t)
	repo := createWorkerActionsRepositoryForTest(pool)
	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)
	worker := seedWorkerForActions(t, ctx, pool, tenantId)

	const total, chunk = 100000, 1000

	prefix := uuid.NewString()
	ids := make([]string, 0, total)

	for i := 0; i < total; i++ {
		ids = append(ids, fmt.Sprintf("%s_scale:action%06d", prefix, i))
	}

	fill := time.Now()

	for start := 0; start < total; start += chunk {
		_, err := addWorkerActions(repo, ctx, tenantId, worker, ids[start:start+chunk])
		require.NoError(t, err)
	}

	t.Logf("filled %d actions in %d chunks: %s", total, total/chunk, time.Since(fill))

	extra := make([]string, 0, chunk)

	for i := 0; i < chunk; i++ {
		extra = append(extra, fmt.Sprintf("%s_scale:extra%04d", prefix, i))
	}

	started := time.Now()
	added, err := addWorkerActions(repo, ctx, tenantId, worker, extra)
	require.NoError(t, err)
	require.Equal(t, chunk, added)
	t.Logf("add chunk of %d new actions at %d linked: %s", chunk, total, time.Since(started))

	started = time.Now()
	added, err = addWorkerActions(repo, ctx, tenantId, worker, ids[:chunk])
	require.NoError(t, err)
	require.Zero(t, added)
	t.Logf("no-op add chunk of %d existing actions at %d linked: %s", chunk, total+chunk, time.Since(started))

	started = time.Now()
	removed, err := removeWorkerActions(repo, ctx, tenantId, worker, extra)
	require.NoError(t, err)
	require.Equal(t, chunk, removed)
	t.Logf("remove chunk of %d actions at %d linked: %s", chunk, total+chunk, time.Since(started))

	assert.Equal(t, hashActions(ids), workerActionHash(t, ctx, pool, worker))
}
