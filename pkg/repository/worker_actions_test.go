//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

func createWorkerActionsRepositoryForTest(pool *pgxpool.Pool) WorkerRepository {
	logger := zerolog.New(io.Discard)

	return newWorkerRepository(&sharedRepository{
		pool:    pool,
		l:       &logger,
		v:       validator.NewDefaultValidator(),
		queries: sqlcv1.New(),
	})
}

// seedWorkerForActions inserts a tenant-owned worker with the empty-set action hash, the same
// starting point CreateNewWorker gives a worker created without actions.
func seedWorkerForActions(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantId uuid.UUID) uuid.UUID {
	t.Helper()

	workerId := uuid.New()

	_, err := pool.Exec(ctx,
		`INSERT INTO "Worker" ("id", "tenantId", "name", "actionHash") VALUES ($1, $2, $3, $4)`,
		workerId, tenantId, "worker-"+workerId.String()[:8], hashActions(nil),
	)
	require.NoError(t, err)

	return workerId
}

func seedTenantForActions(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()

	tenantId := uuid.New()
	slug := "tenant-" + strings.ReplaceAll(tenantId.String(), "-", "")

	_, err := pool.Exec(ctx, `INSERT INTO "Tenant" ("id", "name", "slug") VALUES ($1, $2, $3)`, tenantId, "Test Tenant", slug)
	require.NoError(t, err)

	return tenantId
}

func linkedActions(t *testing.T, ctx context.Context, pool *pgxpool.Pool, workerId uuid.UUID) []string {
	t.Helper()

	rows, err := pool.Query(ctx,
		`SELECT a."actionId" FROM "_ActionToWorker" aw JOIN "Action" a ON a."id" = aw."A" WHERE aw."B" = $1 ORDER BY a."actionId"`,
		workerId,
	)
	require.NoError(t, err)
	defer rows.Close()

	actions := []string{}

	for rows.Next() {
		var action string
		require.NoError(t, rows.Scan(&action))
		actions = append(actions, action)
	}

	require.NoError(t, rows.Err())

	return actions
}

func workerActionHash(t *testing.T, ctx context.Context, pool *pgxpool.Pool, workerId uuid.UUID) []byte {
	t.Helper()

	var hash []byte
	require.NoError(t, pool.QueryRow(ctx, `SELECT "actionHash" FROM "Worker" WHERE "id" = $1`, workerId).Scan(&hash))

	return hash
}

func TestWorkerActionDeltas(t *testing.T) {
	t.Parallel()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	repo := createWorkerActionsRepositoryForTest(pool)
	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)

	t.Run("add and remove round trip", func(t *testing.T) {
		workerId := seedWorkerForActions(t, ctx, pool, tenantId)
		initial := workerActionHash(t, ctx, pool, workerId)

		added, err := repo.AddWorkerActions(ctx, tenantId, workerId, []string{"svc:a", "svc:b", "svc:c"})
		require.NoError(t, err)
		assert.Equal(t, 3, added)
		assert.Equal(t, []string{"svc:a", "svc:b", "svc:c"}, linkedActions(t, ctx, pool, workerId))
		assert.NotEqual(t, initial, workerActionHash(t, ctx, pool, workerId))

		removed, err := repo.RemoveWorkerActions(ctx, tenantId, workerId, []string{"svc:b"})
		require.NoError(t, err)
		assert.Equal(t, 1, removed)
		assert.Equal(t, []string{"svc:a", "svc:c"}, linkedActions(t, ctx, pool, workerId))

		removed, err = repo.RemoveWorkerActions(ctx, tenantId, workerId, []string{"svc:a", "svc:c", "svc:never-added"})
		require.NoError(t, err)
		assert.Equal(t, 2, removed)
		assert.Empty(t, linkedActions(t, ctx, pool, workerId))
		assert.Equal(t, initial, workerActionHash(t, ctx, pool, workerId), "removing every action restores the seed hash")
	})

	t.Run("re-adding is idempotent", func(t *testing.T) {
		workerId := seedWorkerForActions(t, ctx, pool, tenantId)

		added, err := repo.AddWorkerActions(ctx, tenantId, workerId, []string{"svc:a", "svc:b"})
		require.NoError(t, err)
		assert.Equal(t, 2, added)
		hash := workerActionHash(t, ctx, pool, workerId)

		// duplicates within one call and across calls, in any casing, link nothing new
		added, err = repo.AddWorkerActions(ctx, tenantId, workerId, []string{"svc:a", "SVC:B", "svc:b", "svc:c"})
		require.NoError(t, err)
		assert.Equal(t, 1, added)
		assert.Equal(t, []string{"svc:a", "svc:b", "svc:c"}, linkedActions(t, ctx, pool, workerId))
		assert.NotEqual(t, hash, workerActionHash(t, ctx, pool, workerId))

		added, err = repo.AddWorkerActions(ctx, tenantId, workerId, []string{"svc:c"})
		require.NoError(t, err)
		assert.Equal(t, 0, added)

		removed, err := repo.RemoveWorkerActions(ctx, tenantId, workerId, []string{"svc:c"})
		require.NoError(t, err)
		assert.Equal(t, 1, removed)
		assert.Equal(t, hash, workerActionHash(t, ctx, pool, workerId), "the hash only moves for actions actually linked or unlinked")
	})

	t.Run("hash is order independent", func(t *testing.T) {
		first := seedWorkerForActions(t, ctx, pool, tenantId)
		second := seedWorkerForActions(t, ctx, pool, tenantId)

		_, err := repo.AddWorkerActions(ctx, tenantId, first, []string{"svc:a", "svc:b"})
		require.NoError(t, err)
		_, err = repo.AddWorkerActions(ctx, tenantId, first, []string{"svc:c"})
		require.NoError(t, err)

		_, err = repo.AddWorkerActions(ctx, tenantId, second, []string{"svc:c", "svc:b"})
		require.NoError(t, err)
		_, err = repo.AddWorkerActions(ctx, tenantId, second, []string{"svc:a"})
		require.NoError(t, err)

		assert.Equal(t, workerActionHash(t, ctx, pool, first), workerActionHash(t, ctx, pool, second), "the same set built in a different order hashes equal")

		// the shared hash resolves the shared set through the representative-worker lookup
		records, err := sqlcv1.New().GetWorkerActionsByWorkerActionHash(ctx, pool, sqlcv1.GetWorkerActionsByWorkerActionHashParams{
			Tenantid:     tenantId,
			Actionhashes: [][]byte{workerActionHash(t, ctx, pool, first)},
		})
		require.NoError(t, err)

		actions := make([]string, 0, len(records))

		for _, record := range records {
			actions = append(actions, record.ActionID)
		}

		assert.ElementsMatch(t, []string{"svc:a", "svc:b", "svc:c"}, actions)

		_, err = repo.RemoveWorkerActions(ctx, tenantId, second, []string{"svc:b"})
		require.NoError(t, err)
		assert.NotEqual(t, workerActionHash(t, ctx, pool, first), workerActionHash(t, ctx, pool, second), "a removal changes the hash")

		_, err = repo.RemoveWorkerActions(ctx, tenantId, first, []string{"svc:b"})
		require.NoError(t, err)
		assert.Equal(t, workerActionHash(t, ctx, pool, first), workerActionHash(t, ctx, pool, second), "removing the same action from both restores equality")
	})
}
