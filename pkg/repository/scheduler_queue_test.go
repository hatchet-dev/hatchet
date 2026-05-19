//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func createAssignmentRepositoryForTest(pool *pgxpool.Pool) *assignmentRepository {
	logger := zerolog.New(io.Discard)

	return newAssignmentRepository(&sharedRepository{
		pool:    pool,
		l:       &logger,
		queries: sqlcv1.New(),
	})
}

func TestListAvailableSlotsCountsBatchesOnce(t *testing.T) {
	t.Parallel()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	repo := createAssignmentRepositoryForTest(pool)

	ctx := context.Background()

	tenantID := uuid.New()
	workerID := uuid.New()
	batchID := uuid.New()

	now := time.Now().UTC()
	timeoutAt := now.Add(time.Hour)

	slug := "tenant-" + strings.ReplaceAll(tenantID.String(), "-", "")

	_, err := pool.Exec(ctx, `INSERT INTO "Tenant" ("id", "name", "slug") VALUES ($1, $2, $3)`, tenantID, "Test Tenant", slug)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO "Worker" ("id", "tenantId", "name", "maxRuns", "isActive") VALUES ($1, $2, $3, $4, $5)`, workerID, tenantID, "test-worker", 5, true)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, worker_id, tenant_id, timeout_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, int64(1), now, int32(0), workerID, tenantID, timeoutAt)
	require.NoError(t, err)

	for idx, taskID := range []int64{2, 3, 4} {
		_, err = pool.Exec(ctx, `
			INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, worker_id, batch_id, batch_size, batch_index, tenant_id, timeout_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, taskID, now, int32(0), workerID, batchID, int32(3), int32(idx), tenantID, timeoutAt)
		require.NoError(t, err)
	}

	results, err := repo.ListAvailableSlotsForWorkers(ctx, tenantID, sqlcv1.ListAvailableSlotsForWorkersParams{
		Tenantid:  tenantID,
		Workerids: []uuid.UUID{workerID},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, int32(3), results[0].AvailableSlots)
}
