//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func createConcurrencyRepositoryForTest(pool *pgxpool.Pool) *ConcurrencyRepositoryImpl {
	logger := zerolog.New(io.Discard)

	return &ConcurrencyRepositoryImpl{
		sharedRepository: &sharedRepository{
			pool:    pool,
			l:       &logger,
			queries: sqlcv1.New(),
			// upsertQueuesForQueuedTasks (via upsertQueues) reads this cache whenever a poll actually
			// promotes something to Queued - not exercised by every test, but needed once one does.
			queueCache: cache.New(5 * time.Minute),
		},
	}
}

func seedParentChildConcurrency(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, strategy string, maxRuns int32) (workflowID, workflowVersionID uuid.UUID, parentStrategyID, childStrategyID int64) {
	t.Helper()

	workflowID = uuid.New()
	workflowVersionID = uuid.New()

	err := pool.QueryRow(ctx, `
		INSERT INTO v1_step_concurrency (workflow_id, workflow_version_id, step_id, strategy, expression, tenant_id, max_concurrency)
		VALUES ($1, $2, $3, $4::v1_concurrency_strategy, 'input.group', $5, $6)
		RETURNING id
	`, workflowID, workflowVersionID, uuid.New(), strategy, tenantID, maxRuns).Scan(&childStrategyID)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		INSERT INTO v1_workflow_concurrency (workflow_id, workflow_version_id, strategy, expression, tenant_id, max_concurrency, child_strategy_ids)
		VALUES ($1, $2, $3::v1_concurrency_strategy, 'input.group', $4, $5, ARRAY[$6::bigint])
		RETURNING id
	`, workflowID, workflowVersionID, strategy, tenantID, maxRuns, childStrategyID).Scan(&parentStrategyID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `UPDATE v1_step_concurrency SET parent_strategy_id = $1 WHERE id = $2`, parentStrategyID, childStrategyID)
	require.NoError(t, err)

	return workflowID, workflowVersionID, parentStrategyID, childStrategyID
}

func insertParentedConcurrencySlot(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, workflowID, workflowVersionID uuid.UUID, parentStrategyID, childStrategyID, taskID int64, insertedAt time.Time, isFilled bool) uuid.UUID {
	t.Helper()

	workflowRunID := uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO v1_concurrency_slot (
			task_id, task_inserted_at, task_retry_count, external_id, tenant_id,
			workflow_id, workflow_version_id, workflow_run_id,
			strategy_id, parent_strategy_id, priority, key, is_filled,
			queue_to_notify, schedule_timeout_at
		) VALUES ($1, $2, 0, $3, $4, $5, $6, $7, $8, $9, 5, 'test-group', $10, 'default', $11)
	`, taskID, insertedAt, uuid.New(), tenantID, workflowID, workflowVersionID, workflowRunID,
		childStrategyID, parentStrategyID, isFilled, insertedAt.Add(time.Hour))
	require.NoError(t, err)

	if isFilled {
		_, err = pool.Exec(ctx, `
			UPDATE v1_workflow_concurrency_slot SET is_filled = TRUE
			WHERE strategy_id = $1 AND workflow_version_id = $2 AND workflow_run_id = $3
		`, parentStrategyID, workflowVersionID, workflowRunID)
		require.NoError(t, err)
	}

	return workflowRunID
}

func remainingConcurrencySlotTaskIDs(t *testing.T, ctx context.Context, pool *pgxpool.Pool, strategyID int64) []int64 {
	t.Helper()

	rows, err := pool.Query(ctx, `SELECT task_id FROM v1_concurrency_slot WHERE strategy_id = $1 ORDER BY task_id`, strategyID)
	require.NoError(t, err)
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	require.NoError(t, rows.Err())

	return ids
}

func TestCancelExceptNewestWithParentStrategy_SinglePollKeepsNewestQueued(t *testing.T) {
	t.Parallel()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	repo := createConcurrencyRepositoryForTest(pool)

	tenantID := uuid.New()
	now := time.Now().UTC()

	workflowID, workflowVersionID, parentStrategyID, childStrategyID := seedParentChildConcurrency(
		t, ctx, pool, tenantID, "CANCEL_EXCEPT_NEWEST", 1,
	)

	// task 100: already running, occupies the only slot.
	insertParentedConcurrencySlot(t, ctx, pool, tenantID, workflowID, workflowVersionID, parentStrategyID, childStrategyID, 100, now, true)

	// tasks 101..110: arrive one at a time while 100 is still running.
	for i, taskID := range []int64{101, 102, 103, 104, 105, 106, 107, 108, 109, 110} {
		insertParentedConcurrencySlot(t, ctx, pool, tenantID, workflowID, workflowVersionID, parentStrategyID, childStrategyID, taskID, now.Add(time.Duration(i+1)*time.Second), false)
	}

	strategy := &sqlcv1.V1StepConcurrency{
		ID:                childStrategyID,
		ParentStrategyID:  pgtype.Int8{Int64: parentStrategyID, Valid: true},
		WorkflowID:        workflowID,
		WorkflowVersionID: workflowVersionID,
		Strategy:          sqlcv1.V1ConcurrencyStrategyCANCELEXCEPTNEWEST,
		TenantID:          tenantID,
		MaxConcurrency:    1,
	}

	res, err := repo.RunConcurrencyStrategy(ctx, tenantID, strategy)
	require.NoError(t, err)
	require.NotNil(t, res)

	cancelledIDs := make([]int64, 0, len(res.Cancelled))
	for _, c := range res.Cancelled {
		assert.Equal(t, CancelledReasonConcurrencyLimit, c.CancelledReason)
		cancelledIDs = append(cancelledIDs, c.Id)
	}

	assert.ElementsMatch(t, []int64{101, 102, 103, 104, 105, 106, 107, 108, 109}, cancelledIDs,
		"exactly the 9 runs older than the newest queued run should be cancelled")

	remaining := remainingConcurrencySlotTaskIDs(t, ctx, pool, childStrategyID)
	assert.ElementsMatch(t, []int64{100, 110}, remaining,
		"the running run (100) and the newest queued run (110) must both still have a concurrency slot")
}

func TestCancelExceptNewestWithParentStrategy_PromotesSurvivorOnNextPoll(t *testing.T) {
	t.Parallel()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	repo := createConcurrencyRepositoryForTest(pool)

	tenantID := uuid.New()
	now := time.Now().UTC()

	workflowID, workflowVersionID, parentStrategyID, childStrategyID := seedParentChildConcurrency(
		t, ctx, pool, tenantID, "CANCEL_EXCEPT_NEWEST", 1,
	)

	insertParentedConcurrencySlot(t, ctx, pool, tenantID, workflowID, workflowVersionID, parentStrategyID, childStrategyID, 200, now, true)
	insertParentedConcurrencySlot(t, ctx, pool, tenantID, workflowID, workflowVersionID, parentStrategyID, childStrategyID, 201, now.Add(time.Second), false)
	survivorRunID := insertParentedConcurrencySlot(t, ctx, pool, tenantID, workflowID, workflowVersionID, parentStrategyID, childStrategyID, 202, now.Add(2*time.Second), false)

	strategy := &sqlcv1.V1StepConcurrency{
		ID:                childStrategyID,
		ParentStrategyID:  pgtype.Int8{Int64: parentStrategyID, Valid: true},
		WorkflowID:        workflowID,
		WorkflowVersionID: workflowVersionID,
		Strategy:          sqlcv1.V1ConcurrencyStrategyCANCELEXCEPTNEWEST,
		TenantID:          tenantID,
		MaxConcurrency:    1,
	}

	// first poll: 201 should be cancelled, 202 (newest) should survive queued, 200 keeps running.
	_, err := repo.RunConcurrencyStrategy(ctx, tenantID, strategy)
	require.NoError(t, err)

	remaining := remainingConcurrencySlotTaskIDs(t, ctx, pool, childStrategyID)
	require.ElementsMatch(t, []int64{200, 202}, remaining, "201 should already be cancelled after the first poll")

	// simulate task 200 completing: its concurrency slot and admission-gate row are released, exactly
	// as after_v1_concurrency_slot_delete_function() would do for a real task completion.
	_, err = pool.Exec(ctx, `DELETE FROM v1_concurrency_slot WHERE strategy_id = $1 AND task_id = 200`, childStrategyID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `DELETE FROM v1_workflow_concurrency_slot WHERE strategy_id = $1 AND workflow_run_id != $2`, parentStrategyID, survivorRunID)
	require.NoError(t, err)

	// second poll: the slot is free, so the survivor (202) must be promoted to running now - not on
	// some later poll, and without needing a new arrival on this key.
	res, err := repo.RunConcurrencyStrategy(ctx, tenantID, strategy)
	require.NoError(t, err)
	require.NotNil(t, res)

	queuedIDs := make([]int64, 0, len(res.Queued))
	for _, q := range res.Queued {
		queuedIDs = append(queuedIDs, q.Id)
	}

	assert.ElementsMatch(t, []int64{202}, queuedIDs, "the survivor must be promoted as soon as the slot frees")
}
