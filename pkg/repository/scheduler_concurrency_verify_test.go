//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// Throwaway verification for the RunCancelExceptNewest SQL fix (no-parent branch). Exercises the
// query directly rather than through runCancelExceptNewest, since that requires a v1_step_concurrency
// row to exist too (harmless either way, but the query itself is what changed).
func TestVerify_RunCancelExceptNewest_KeepsNewestQueued(t *testing.T) {
	t.Parallel()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	queries := sqlcv1.New()

	tenantID := uuid.New()
	workflowID := uuid.New()
	workflowVersionID := uuid.New()
	now := time.Now().UTC()

	var strategyID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO v1_step_concurrency (workflow_id, workflow_version_id, step_id, strategy, expression, tenant_id, max_concurrency)
		VALUES ($1, $2, $3, 'CANCEL_EXCEPT_NEWEST', 'input.group', $4, 1)
		RETURNING id
	`, workflowID, workflowVersionID, uuid.New(), tenantID).Scan(&strategyID)
	require.NoError(t, err)

	insert := func(taskID int64, insertedAt time.Time, isFilled bool) {
		_, err := pool.Exec(ctx, `
			INSERT INTO v1_concurrency_slot (
				task_id, task_inserted_at, task_retry_count, external_id, tenant_id,
				workflow_id, workflow_version_id, workflow_run_id,
				strategy_id, priority, key, is_filled, queue_to_notify, schedule_timeout_at
			) VALUES ($1, $2, 0, $3, $4, $5, $6, $7, $8, 5, 'test-group', $9, 'default', $10)
		`, taskID, insertedAt, uuid.New(), tenantID, workflowID, workflowVersionID, uuid.New(),
			strategyID, isFilled, insertedAt.Add(time.Hour))
		require.NoError(t, err)
	}

	insert(100, now, true) // already running
	for i, taskID := range []int64{101, 102, 103, 104, 105} {
		insert(taskID, now.Add(time.Duration(i+1)*time.Second), false)
	}

	rows, err := queries.RunCancelExceptNewest(ctx, pool, sqlcv1.RunCancelExceptNewestParams{
		Tenantid:   tenantID,
		Strategyid: strategyID,
		Maxruns:    1,
	})
	require.NoError(t, err)

	var cancelled []int64
	for _, r := range rows {
		if r.Operation == "CANCELLED" {
			cancelled = append(cancelled, r.TaskID)
		}
	}

	assert.ElementsMatch(t, []int64{101, 102, 103, 104}, cancelled, "everyone except the running task and the newest queued task (105) should be cancelled")

	remaining := remainingConcurrencySlotTaskIDs(t, ctx, pool, strategyID)
	assert.ElementsMatch(t, []int64{100, 105}, remaining, "the running task and the newest queued task must both still have a concurrency slot")
}

// Verification for the RunCancelExceptOldest SQL (no-parent branch), mirroring the newest test above.
func TestVerify_RunCancelExceptOldest_KeepsOldestQueued(t *testing.T) {
	t.Parallel()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	queries := sqlcv1.New()

	tenantID := uuid.New()
	workflowID := uuid.New()
	workflowVersionID := uuid.New()
	now := time.Now().UTC()

	var strategyID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO v1_step_concurrency (workflow_id, workflow_version_id, step_id, strategy, expression, tenant_id, max_concurrency)
		VALUES ($1, $2, $3, 'CANCEL_EXCEPT_OLDEST', 'input.group', $4, 1)
		RETURNING id
	`, workflowID, workflowVersionID, uuid.New(), tenantID).Scan(&strategyID)
	require.NoError(t, err)

	insert := func(taskID int64, insertedAt time.Time, isFilled bool) {
		_, err := pool.Exec(ctx, `
			INSERT INTO v1_concurrency_slot (
				task_id, task_inserted_at, task_retry_count, external_id, tenant_id,
				workflow_id, workflow_version_id, workflow_run_id,
				strategy_id, priority, key, is_filled, queue_to_notify, schedule_timeout_at
			) VALUES ($1, $2, 0, $3, $4, $5, $6, $7, $8, 5, 'test-group', $9, 'default', $10)
		`, taskID, insertedAt, uuid.New(), tenantID, workflowID, workflowVersionID, uuid.New(),
			strategyID, isFilled, insertedAt.Add(time.Hour))
		require.NoError(t, err)
	}

	insert(100, now, true) // already running
	for i, taskID := range []int64{101, 102, 103, 104, 105} {
		insert(taskID, now.Add(time.Duration(i+1)*time.Second), false)
	}

	rows, err := queries.RunCancelExceptOldest(ctx, pool, sqlcv1.RunCancelExceptOldestParams{
		Tenantid:   tenantID,
		Strategyid: strategyID,
		Maxruns:    1,
	})
	require.NoError(t, err)

	var cancelled []int64
	for _, r := range rows {
		if r.Operation == "CANCELLED" {
			cancelled = append(cancelled, r.TaskID)
		}
	}

	assert.ElementsMatch(t, []int64{102, 103, 104, 105}, cancelled, "everyone except the running task and the oldest queued task (101) should be cancelled")

	remaining := remainingConcurrencySlotTaskIDs(t, ctx, pool, strategyID)
	assert.ElementsMatch(t, []int64{100, 101}, remaining, "the running task and the oldest queued task must both still have a concurrency slot")
}
