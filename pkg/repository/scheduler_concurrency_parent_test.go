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

// This file exercises the OLD (SQL-based) scheduler's parent/child concurrency mechanism -
// ParentStrategyID / RunParentX / RunChildX in scheduler_concurrency.go - directly against a real
// Postgres database.
//
// This is the ONLY way to reach that code path today. It is NOT reachable via any modern SDK
// (Python, TypeScript, Go v1, Ruby): the v1 admin server's CreateStepConcurrency insert
// (pkg/repository/sqlcv1/workflows.sql, "CreateStepConcurrency") never sets parent_strategy_id.
// parent_strategy_id is only ever populated by the legacy create_v1_step_concurrency() trigger
// (sql/schema/v1-core.sql), which fires off the deprecated v0 Workflow.concurrencyGroupExpression
// column - a path no current SDK's `hatchet.workflow(concurrency=...)` call reaches, since it always
// goes through the v1 CreateWorkflowVersionRequest (see runnables/workflow.py: even a single
// ConcurrencyExpression is sent via the deprecated singular `concurrency` proto field, but
// internal/services/admin/v1/server.go merges it into the same ordered `concurrency []CreateConcurrencyOpts`
// chain as concurrency_arr, and pkg/repository/workflow.go's CreateStepConcurrency never sets
// parent_strategy_id for any row created that way).
//
// Concretely: with a *single-task* workflow, the sdks/python/examples/concurrency_cancel_except_newest_with_parent_concurrency
// example does NOT exercise this file's code path despite the name -
// pkg/repository/workflow.go:mergeWorkflowConcurrencyOntoSingleTask folds workflow-level concurrency
// onto the sole task instead, leaving parent_strategy_id NULL. With >1 task the merge doesn't fire and
// the workflow's own declared strategy (only that one - a separate task-level `concurrency=[...]`
// array is always independent, parent_strategy_id NULL regardless of task count) does get a real
// parent_strategy_id, exercising this file's code path end to end.

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

// seedParentChildConcurrency creates a real v1_workflow_concurrency ("parent") row and a matching
// v1_step_concurrency ("child") row with parent_strategy_id set - the shape the legacy
// create_v1_step_concurrency() trigger produces, and the only shape that ever exercises
// strategy.ParentStrategyID.Valid in scheduler_concurrency.go. The trigger always mirrors the same
// max_concurrency onto both rows (sql/schema/v1-core.sql: `NEW."maxRuns"` is used for both the
// parent INSERT and every child INSERT), and RunConcurrencyStrategy's parent-branch calls
// (RunParentGroupRoundRobin, RunParentCancelInProgress, RunParentCancelNewest, ...) all pass the
// CHILD strategy's MaxConcurrency as the parent admission limit - never the parent row's own stored
// max_concurrency - so this fixture keeps both values equal, matching that invariant.
func seedParentChildConcurrency(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, strategy string, maxRuns int32) (workflowID, workflowVersionID uuid.UUID, parentStrategyID, childStrategyID int64) {
	t.Helper()

	workflowID = uuid.New()
	workflowVersionID = uuid.New()

	// The child row must exist before the parent row, because v1_workflow_concurrency.child_strategy_ids
	// (populated by the real create_v1_step_concurrency() trigger with the ids of the child rows it just
	// created) cannot be left NULL or empty here: after_v1_concurrency_slot_insert_function's
	// ARRAY_AGG(DISTINCT wc.child_strategy_ids) rejects both a null array ("cannot accumulate null
	// arrays") and an empty array ("cannot accumulate empty arrays") when aggregating this BIGINT[]
	// column, since it can't determine the resulting array's dimensionality. So: create the child
	// first (parent_strategy_id NULL), then the parent (with a real, non-empty child_strategy_ids),
	// then link the child to it.
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

// insertParentedConcurrencySlot inserts one task-level concurrency slot for a brand-new workflow
// run, gated by the given parent+child strategy pair - mirroring one `aio_run()` call in the Python
// "with_parent_concurrency" examples, except with a real parent_strategy_id so the fixture actually
// reaches the code under test. Inserting with parent_strategy_id set fires
// after_v1_concurrency_slot_insert_function(), which auto-creates the matching (initially unfilled)
// v1_workflow_concurrency_slot admission-gate row for this run.
//
// If isFilled is true, both the task-level slot and its just-created parent admission-gate row are
// marked filled, simulating a run that was already admitted and is actively executing (i.e. the
// "occupying" run in the Python examples).
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

// TestCancelExceptNewestWithParentStrategy_SinglePollKeepsNewestQueued is a direct repro of the gap
// this file documents above: RunCancelExceptNewest's parent branch (runCancelExceptNewest in
// scheduler_concurrency.go) currently calls RunParentCancelNewest/RunChildCancelNewest - plain
// CANCEL_NEWEST semantics - because RunParentCancelExceptNewest/RunChildCancelExceptNewest don't
// exist yet. Under CANCEL_NEWEST, every queued run that doesn't fit is cancelled outright, including
// the newest one. CANCEL_EXCEPT_NEWEST must instead leave the single newest queued run alone (still
// queued, to be promoted once the running slot frees), cancelling only the runs strictly older than
// it (and older than what's already running).
//
// One task (100) is seeded already running (both its parent and child slots filled). Ten more
// (101..110) are seeded queued, in arrival order, competing for the same maxRuns=1 group. A single
// call to RunConcurrencyStrategy must cancel 101..109 and leave the running task (100) and the
// newest queued task (110) untouched.
//
// This currently FAILS: task 110 gets cancelled along with the rest, because the parent branch is
// still wired to CANCEL_NEWEST. It will pass once RunParentCancelExceptNewest/RunChildCancelExceptNewest
// are implemented with real "except newest" semantics.
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

// TestCancelExceptNewestWithParentStrategy_PromotesSurvivorOnNextPoll extends the above: once the
// occupying run's slot frees, the surviving newest queued run must be promoted on the very next
// RunConcurrencyStrategy call - it should not need a fresh arrival on this key to be reconsidered.
//
// This currently fails at the same assertion as the single-poll test above (task 110 never survives
// to be promoted, because it's cancelled on the first call).
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
