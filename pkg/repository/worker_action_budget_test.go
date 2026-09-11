//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// seedOperator inserts a GRPC operator row for the tenant.
func seedOperator(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantId uuid.UUID) uuid.UUID {
	t.Helper()

	operatorId := uuid.New()

	_, err := pool.Exec(ctx,
		`INSERT INTO v1_operator (id, tenant_id, name, kind, config) VALUES ($1, $2, $3, 'GRPC', '{}'::jsonb)`,
		operatorId, tenantId, "op-"+operatorId.String()[:8],
	)
	require.NoError(t, err)

	return operatorId
}

// seedOperatorWorker inserts a worker of the operator with the empty-set hash and no links.
func seedOperatorWorker(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantId, operatorId uuid.UUID) uuid.UUID {
	t.Helper()

	workerId := uuid.New()

	_, err := pool.Exec(ctx,
		`INSERT INTO "Worker" ("id", "tenantId", "name", "actionHash", "operatorId") VALUES ($1, $2, $3, $4, $5)`,
		workerId, tenantId, "worker-"+workerId.String()[:8], hashActions(nil), operatorId,
	)
	require.NoError(t, err)

	return workerId
}

// workerActionCount reads the count the worker row carries.
func workerActionCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, workerId uuid.UUID) int {
	t.Helper()

	var n int
	require.NoError(t, pool.QueryRow(ctx, `SELECT "actionCount" FROM "Worker" WHERE "id" = $1`, workerId).Scan(&n))

	return n
}

func ids(prefix string, from, to int) []string {
	out := make([]string, 0, to-from)

	for i := from; i < to; i++ {
		out = append(out, fmt.Sprintf("svc:%s%d", prefix, i))
	}

	return out
}

// The action budget is the operator's: two of its workers fill it together, the delta past it
// is refused with the operator's totals and rolled back whole, and a removal on one worker
// frees room for the other. "actionCount" follows the links on every worker.
func TestOperatorActionBudgetSpansWorkers(t *testing.T) {
	pool := workerActionsPool(t)
	repo := createWorkerActionsRepositoryForTest(pool)
	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)
	operatorId := seedOperator(t, ctx, pool, tenantId)
	first := seedOperatorWorker(t, ctx, pool, tenantId, operatorId)
	second := seedOperatorWorker(t, ctx, pool, tenantId, operatorId)

	const cap = 30

	for i, worker := range []uuid.UUID{first, second, first} {
		added, _, err := repo.ApplyWorkerActionsDelta(ctx, tenantId, worker, ids("a", i*10, (i+1)*10), nil, cap)
		require.NoError(t, err, "chunk %d fits", i)
		assert.Equal(t, 10, added)
	}

	added, removed, err := repo.ApplyWorkerActionsDelta(ctx, tenantId, second, ids("a", 30, 31), []string{"svc:a10"}, cap)
	require.NoError(t, err, "a delta that adds and removes one stays at the cap")
	assert.Equal(t, 1, added)
	assert.Equal(t, 1, removed)

	_, _, err = repo.ApplyWorkerActionsDelta(ctx, tenantId, second, ids("b", 0, 2), []string{"svc:a11"}, cap)
	require.ErrorIs(t, err, ErrWorkerActionBudgetExceeded)

	var budgetErr *ActionBudgetError
	require.True(t, errors.As(err, &budgetErr))
	assert.Equal(t, operatorId, budgetErr.OperatorId)
	assert.EqualValues(t, 31, budgetErr.Linked, "the totals are what the delta would have left")
	assert.EqualValues(t, cap, budgetErr.Limit)

	assert.Len(t, linkedActions(t, ctx, pool, second), 10, "the refused delta rolled back its removes with its adds")
	assert.Equal(t, 20, workerActionCount(t, ctx, pool, first))
	assert.Equal(t, 10, workerActionCount(t, ctx, pool, second))

	total, err := repo.CountOperatorWorkerActions(ctx, tenantId, operatorId)
	require.NoError(t, err)
	assert.EqualValues(t, cap, total)

	// a removal on one worker frees room for the other
	_, removed, err = repo.ApplyWorkerActionsDelta(ctx, tenantId, first, nil, ids("a", 0, 3), cap)
	require.NoError(t, err)
	assert.Equal(t, 3, removed)
	assert.Equal(t, 17, workerActionCount(t, ctx, pool, first))

	added, _, err = repo.ApplyWorkerActionsDelta(ctx, tenantId, second, ids("b", 0, 3), nil, cap)
	require.NoError(t, err)
	assert.Equal(t, 3, added)
	assert.Equal(t, 13, workerActionCount(t, ctx, pool, second))

	_, _, err = repo.ApplyWorkerActionsDelta(ctx, tenantId, second, ids("b", 3, 4), nil, cap)
	require.ErrorIs(t, err, ErrWorkerActionBudgetExceeded)

	// a negative cap is no cap, and a worker of no operator is never capped
	_, _, err = repo.ApplyWorkerActionsDelta(ctx, tenantId, second, ids("b", 3, 4), nil, -1)
	require.NoError(t, err)

	loose := seedWorkerForActions(t, ctx, pool, tenantId)
	added, _, err = repo.ApplyWorkerActionsDelta(ctx, tenantId, loose, ids("c", 0, 5), nil, 1)
	require.NoError(t, err)
	assert.Equal(t, 5, added)
	assert.Equal(t, 5, workerActionCount(t, ctx, pool, loose))
}

// Concurrent deltas on two workers of one operator never leave the operator over its cap, and
// every worker's count is its real link count afterwards.
func TestOperatorActionBudgetUnderConcurrentDeltas(t *testing.T) {
	pool := workerActionsPool(t)
	repo := createWorkerActionsRepositoryForTest(pool)
	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)
	operatorId := seedOperator(t, ctx, pool, tenantId)
	workers := []uuid.UUID{seedOperatorWorker(t, ctx, pool, tenantId, operatorId), seedOperatorWorker(t, ctx, pool, tenantId, operatorId)}

	const cap = 120
	const callers, perCaller = 8, 40

	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted, refused := 0, 0

	for i := 0; i < callers; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			for j := 0; j < perCaller; j++ {
				_, _, err := repo.ApplyWorkerActionsDelta(ctx, tenantId, workers[i%2], []string{fmt.Sprintf("svc:c%d-%d", i, j)}, nil, cap)

				mu.Lock()

				switch {
				case err == nil:
					accepted++
				case errors.Is(err, ErrWorkerActionBudgetExceeded):
					refused++
				default:
					t.Errorf("delta %d-%d: %v", i, j, err)
				}

				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, cap, accepted, "exactly the cap is accepted")
	assert.Equal(t, callers*perCaller-cap, refused)

	total := 0

	for _, worker := range workers {
		links := len(linkedActions(t, ctx, pool, worker))
		assert.Equal(t, links, workerActionCount(t, ctx, pool, worker), "the count is the link count")
		total += links
	}

	assert.Equal(t, cap, total)

	sum, err := repo.CountOperatorWorkerActions(ctx, tenantId, operatorId)
	require.NoError(t, err)
	assert.EqualValues(t, cap, sum)
}

// A delta leaves the worker without a hash; the refresh writes the digest of the links the
// deltas left behind, computed the same way in Go and in SQL.
func TestWorkerActionHashRefreshFollowsDeltas(t *testing.T) {
	pool := workerActionsPool(t)
	repo := createWorkerActionsRepositoryForTest(pool)
	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)
	worker := seedWorkerForActions(t, ctx, pool, tenantId)

	_, err := addWorkerActions(repo, ctx, tenantId, worker, []string{"svc:a", "svc:b"})
	require.NoError(t, err)
	assert.Nil(t, workerActionHash(t, ctx, pool, worker), "a delta clears the hash until the refresh")

	_, err = addWorkerActions(repo, ctx, tenantId, worker, []string{"svc:c"})
	require.NoError(t, err)
	_, err = removeWorkerActions(repo, ctx, tenantId, worker, []string{"svc:b"})
	require.NoError(t, err)

	require.NoError(t, repo.RefreshWorkerActionHash(ctx, tenantId, worker))
	assert.Equal(t, hashActions([]string{"svc:a", "svc:c"}), workerActionHash(t, ctx, pool, worker))
	assert.Equal(t, 2, workerActionCount(t, ctx, pool, worker))

	// a refresh with nothing pending is idempotent
	require.NoError(t, repo.RefreshWorkerActionHash(ctx, tenantId, worker))
	assert.Equal(t, hashActions([]string{"svc:a", "svc:c"}), workerActionHash(t, ctx, pool, worker))

	other := seedTenantForActions(t, ctx, pool)
	require.Error(t, repo.RefreshWorkerActionHash(ctx, other, worker), "another tenant cannot refresh the worker")
}

// The Go and SQL digests agree byte for byte for a fixed set of action sets, including ids
// whose concatenation is the same bytes, mixed casing and the empty set, and distinct sets
// hash distinct.
func TestWorkerActionHashGoAndSQLAgree(t *testing.T) {
	pool := workerActionsPool(t)
	repo := createWorkerActionsRepositoryForTest(pool)
	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)

	sets := [][]string{
		{"svc:a", "svc:b"},
		{"svc:ab", "svc:c"},
		{"svc:a", "svc:bc"},
		{"svc:a", "svc:b", "svc:c"},
		{},
	}

	hashes := make(map[string][]string, len(sets))

	for _, set := range sets {
		worker := seedWorkerForActions(t, ctx, pool, tenantId)

		if len(set) > 0 {
			_, err := addWorkerActions(repo, ctx, tenantId, worker, set)
			require.NoError(t, err)
			require.NoError(t, repo.RefreshWorkerActionHash(ctx, tenantId, worker))
		}

		stored := workerActionHash(t, ctx, pool, worker)
		assert.Equal(t, hashActions(set), stored, "Go and SQL agree on %v", set)

		fromSQL, err := sqlcv1.New().ComputeWorkerActionHash(ctx, pool, worker)
		require.NoError(t, err)
		assert.Equal(t, stored, fromSQL)

		if previous, ok := hashes[string(stored)]; ok {
			t.Errorf("sets %v and %v hash equal", previous, set)
		}

		hashes[string(stored)] = set
	}

	// casing, order and duplicates normalise the same way on both sides
	worker := seedWorkerForActions(t, ctx, pool, tenantId)
	_, err := addWorkerActions(repo, ctx, tenantId, worker, []string{"svc:b", "svc:A", "svc:a"})
	require.NoError(t, err)
	require.NoError(t, repo.RefreshWorkerActionHash(ctx, tenantId, worker))
	assert.Equal(t, hashActions([]string{"svc:a", "svc:b"}), workerActionHash(t, ctx, pool, worker))
}

// The pause is fenced on the listener session: a superseded session's pause changes nothing
// and reports pgx.ErrNoRows, while the live session's pause lands.
func TestPauseWorkerForListenerIsFenced(t *testing.T) {
	pool := workerActionsPool(t)
	repo := createWorkerActionsRepositoryForTest(pool)
	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)
	worker := seedWorkerForActions(t, ctx, pool, tenantId)

	isPaused := func() bool {
		var paused bool
		require.NoError(t, pool.QueryRow(ctx, `SELECT "isPaused" FROM "Worker" WHERE "id" = $1`, worker).Scan(&paused))

		return paused
	}

	first, second := uuid.New(), uuid.New()

	_, err := repo.ActivateWorkerListener(ctx, tenantId, worker, first)
	require.NoError(t, err)
	require.NoError(t, repo.PauseWorkerForListener(ctx, tenantId, worker, first, true))
	assert.True(t, isPaused())
	require.NoError(t, repo.PauseWorkerForListener(ctx, tenantId, worker, first, false))
	assert.False(t, isPaused())

	_, err = repo.ActivateWorkerListener(ctx, tenantId, worker, second)
	require.NoError(t, err)

	err = repo.PauseWorkerForListener(ctx, tenantId, worker, first, true)
	require.ErrorIs(t, err, pgx.ErrNoRows, "a superseded session cannot pause")
	assert.False(t, isPaused())

	other := seedTenantForActions(t, ctx, pool)
	require.ErrorIs(t, repo.PauseWorkerForListener(ctx, other, worker, second, true), pgx.ErrNoRows, "another tenant cannot pause")

	require.NoError(t, repo.PauseWorkerForListener(ctx, tenantId, worker, second, true))
	assert.True(t, isPaused())
}
