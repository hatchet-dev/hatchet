//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// TestManualSlotRelease checks capacity in the real slot table while the task
// remains running, including repeated release and cross-tenant isolation.
func TestManualSlotRelease(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := t.Context()
	repo := newBatchTestRepository(pool)
	task := insertTaskWithRetries(t, ctx, repo, "manual-slot-release", 1)
	worker := uuid.New()

	_, err := pool.Exec(ctx, `INSERT INTO v1_task_runtime
		(task_id, task_inserted_at, retry_count, worker_id, tenant_id, timeout_at)
		VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '1 hour')`,
		task.ID, task.InsertedAt, task.RetryCount, worker, internalTenantId)
	require.NoError(t, err)

	// Slot usage can span several resource types and more than one unit per type.
	_, err = pool.Exec(ctx, `INSERT INTO v1_task_runtime_slot
		(tenant_id, task_id, task_inserted_at, retry_count, worker_id, slot_type, units)
		VALUES ($1,$2,$3,$4,$5,'default',2), ($1,$2,$3,$4,$5,'memory',3)
		ON CONFLICT (task_id,task_inserted_at,retry_count,slot_type) DO UPDATE SET units=EXCLUDED.units`,
		internalTenantId, task.ID, task.InsertedAt, task.RetryCount, worker)
	require.NoError(t, err)

	// Knowing the external task ID must not allow another tenant to release it.
	_, err = repo.ReleaseSlot(ctx, uuid.New(), task.ExternalID)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	var slots int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM v1_task_runtime_slot WHERE task_id=$1`, task.ID).Scan(&slots))
	require.Equal(t, 2, slots)

	for release := 0; release < 2; release++ {
		runtime, err := repo.ReleaseSlot(ctx, internalTenantId, task.ExternalID)
		require.NoError(t, err)
		require.Equal(t, task.ID, runtime.TaskID)
		require.Nil(t, runtime.WorkerID)

		// A successful return and a NULL worker_id do not prove capacity is free.
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM v1_task_runtime_slot WHERE task_id=$1`, task.ID).Scan(&slots))
		require.Zero(t, slots, "release %d must remove all slot usage", release+1)

		// Releasing capacity must retain the runtime record for task completion.
		var active int
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM v1_task_runtime WHERE task_id=$1`, task.ID).Scan(&active))
		require.Equal(t, 1, active)
	}
}
