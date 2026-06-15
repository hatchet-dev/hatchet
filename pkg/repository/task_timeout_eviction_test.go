//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// TestListTasksToTimeout_ExcludesEvictedRuntimes verifies that evicted
// tasks are not selected for timeout processing.
func TestListTasksToTimeout_ExcludesEvictedRuntimes(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tenantID := uuid.New()
	workerID := uuid.New()
	stepID := uuid.New()
	workflowID := uuid.New()
	workflowVersionID := uuid.New()
	workflowRunID := uuid.New()

	now := time.Now().UTC()
	pastTimeout := now.Add(-5 * time.Minute)

	_, err := pool.Exec(ctx, `
		INSERT INTO "Tenant" ("id", "name", "slug", "createdAt", "updatedAt")
		VALUES ($1, 'test-tenant', 'test-tenant', NOW(), NOW())
	`, tenantID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO "Worker" ("id", "tenantId", "name", "lastHeartbeatAt", "createdAt", "updatedAt")
		VALUES ($1, $2, 'test-worker', NOW(), NOW(), NOW())
	`, workerID, tenantID)
	require.NoError(t, err)

	// Insert two v1_task rows: one for the evicted runtime, one for the non-evicted runtime.
	// Use OVERRIDING SYSTEM VALUE to set the identity column.
	_, err = pool.Exec(ctx, `
		INSERT INTO v1_task (id, inserted_at, tenant_id, queue, action_id, step_id, step_readable_id,
			workflow_id, workflow_version_id, workflow_run_id, schedule_timeout,
			step_timeout, sticky, external_id, display_name, input, step_index)
		OVERRIDING SYSTEM VALUE
		VALUES
			(1, $1, $2, 'default', 'test-action', $3, 'test-step', $4, $5, $6, '5m', '30s', 'NONE', $7, 'test-task-1', '{}', 0),
			(2, $1, $2, 'default', 'test-action', $3, 'test-step', $4, $5, $6, '5m', '30s', 'NONE', $8, 'test-task-2', '{}', 1)
	`, now, tenantID, stepID, workflowID, workflowVersionID, workflowRunID, uuid.New(), uuid.New())
	require.NoError(t, err)

	// Insert two v1_task_runtime rows, both with timeout_at in the past:
	// - Task 1: evicted and should NOT be returned
	// - Task 2: not evicted and SHOULD be returned
	_, err = pool.Exec(ctx, `
		INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, worker_id, tenant_id, timeout_at, evicted_at)
		VALUES
			(1, $1, 0, $2, $3, $4, $5),
			(2, $1, 0, $2, $3, $4, NULL)
	`, now, workerID, tenantID, pastTimeout, now.Add(-3*time.Minute))
	require.NoError(t, err)

	queries := sqlcv1.New()

	results, err := queries.ListTasksToTimeout(ctx, pool, sqlcv1.ListTasksToTimeoutParams{
		Tenantid: tenantID,
		Limit:    pgtype.Int4{Int32: 100, Valid: true},
	})
	require.NoError(t, err)

	// Only the non-evicted runtime (task_id=2) should be returned
	assert.Len(t, results, 1, "should return exactly one task (the non-evicted one)")
	if len(results) == 1 {
		assert.Equal(t, int64(2), results[0].ID, "returned task should be the non-evicted one (id=2)")
	}
}
