//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

func createOperatorRepositoryForTest(pool *pgxpool.Pool) OperatorRepository {
	logger := zerolog.New(io.Discard)

	return newOperatorRepository(&sharedRepository{
		pool:    pool,
		l:       &logger,
		v:       validator.NewDefaultValidator(),
		queries: sqlcv1.New(),
	})
}

// storedWorkerActionHash reads the hash the worker row carries.
func storedWorkerActionHash(t *testing.T, ctx context.Context, pool *pgxpool.Pool, workerId uuid.UUID) []byte {
	t.Helper()

	var hash []byte
	require.NoError(t, pool.QueryRow(ctx, `SELECT "actionHash" FROM "Worker" WHERE "id" = $1`, workerId).Scan(&hash))

	return hash
}

// linkedWorkerActionIds lists the action ids the worker is linked to.
func linkedWorkerActionIds(t *testing.T, ctx context.Context, pool *pgxpool.Pool, workerId uuid.UUID) []string {
	t.Helper()

	rows, err := pool.Query(ctx, `SELECT a."actionId" FROM "_ActionToWorker" aw JOIN "Action" a ON a."id" = aw."A" WHERE aw."B" = $1`, workerId)
	require.NoError(t, err)
	defer rows.Close()

	var ids []string

	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}

	require.NoError(t, rows.Err())

	return ids
}

// The stored hash is the canonical digest of the links the call leaves behind, whether the
// worker already held some of them or none, and never the digest of the list that was given.
func TestUpdateOperatorWorkerActionsHashIsDigestOfLinkedSet(t *testing.T) {
	pool := workerActionsPool(t)
	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)
	workerId := seedWorkerForActions(t, ctx, pool, tenantId)
	workers := createWorkerActionsRepositoryForTest(pool)
	operators := createOperatorRepositoryForTest(pool)

	_, err := workers.AddWorkerActions(ctx, tenantId, workerId, []string{"svc:existing"})
	require.NoError(t, err)

	require.NoError(t, operators.UpdateOperatorWorkerActions(ctx, tenantId, workerId, []string{"Svc:Added", "svc:existing", "svc:other", "svc:other"}))

	linked := linkedWorkerActionIds(t, ctx, pool, workerId)
	assert.ElementsMatch(t, []string{"svc:added", "svc:existing", "svc:other"}, linked)

	stored := storedWorkerActionHash(t, ctx, pool, workerId)
	assert.Equal(t, hashActions(linked), stored, "the hash is the digest of the final linked set")
	assert.NotEqual(t, hashActions([]string{"Svc:Added", "svc:existing", "svc:other", "svc:other"}), stored[:0], "sanity")

	fromSQL, err := sqlcv1.New().ComputeWorkerActionHash(ctx, pool, workerId)
	require.NoError(t, err)
	assert.Equal(t, fromSQL, stored)

	// a call that names a subset of what is linked keeps the hash at the full set's digest,
	// where the previous implementation stored the digest of the subset
	require.NoError(t, operators.UpdateOperatorWorkerActions(ctx, tenantId, workerId, []string{"svc:other"}))
	assert.Equal(t, hashActions(linked), storedWorkerActionHash(t, ctx, pool, workerId))
	assert.NotEqual(t, hashActions([]string{"svc:other"}), storedWorkerActionHash(t, ctx, pool, workerId))

	// the worker repository's delta path and this call agree on the same set
	sibling := seedWorkerForActions(t, ctx, pool, tenantId)
	_, err = workers.AddWorkerActions(ctx, tenantId, sibling, []string{"svc:added", "svc:existing", "svc:other"})
	require.NoError(t, err)
	assert.Equal(t, storedWorkerActionHash(t, ctx, pool, sibling), storedWorkerActionHash(t, ctx, pool, workerId))
}

// A worker of another tenant is not touched.
func TestUpdateOperatorWorkerActionsChecksTenant(t *testing.T) {
	pool := workerActionsPool(t)
	ctx := context.Background()
	tenantId := seedTenantForActions(t, ctx, pool)
	otherTenant := seedTenantForActions(t, ctx, pool)
	workerId := seedWorkerForActions(t, ctx, pool, tenantId)
	operators := createOperatorRepositoryForTest(pool)

	err := operators.UpdateOperatorWorkerActions(ctx, otherTenant, workerId, []string{"svc:run"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong to tenant")

	assert.Empty(t, linkedWorkerActionIds(t, ctx, pool, workerId))
	assert.Equal(t, hashActions(nil), storedWorkerActionHash(t, ctx, pool, workerId))
}
