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

// Mirrors scheduler_concurrency_parent_test.go's CANCEL_EXCEPT_NEWEST coverage, for
// CANCEL_EXCEPT_OLDEST's parent branch (RunParentCancelExceptOldest/RunChildCancelExceptOldest).
// See that file's header comment for why a hand-built parent_strategy_id fixture is the only way to
// reach this code path at all.

// TestCancelExceptOldestWithParentStrategy_SinglePollKeepsOldestQueued mirrors
// TestCancelExceptNewestWithParentStrategy_SinglePollKeepsNewestQueued, but the survivor is the
// oldest queued task (101) instead of the newest (110).
func TestCancelExceptOldestWithParentStrategy_SinglePollKeepsOldestQueued(t *testing.T) {
	t.Parallel()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	repo := createConcurrencyRepositoryForTest(pool)

	tenantID := uuid.New()
	now := time.Now().UTC()

	workflowID, workflowVersionID, parentStrategyID, childStrategyID := seedParentChildConcurrency(
		t, ctx, pool, tenantID, "CANCEL_EXCEPT_OLDEST", 1,
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
		Strategy:          sqlcv1.V1ConcurrencyStrategyCANCELEXCEPTOLDEST,
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

	assert.ElementsMatch(t, []int64{102, 103, 104, 105, 106, 107, 108, 109, 110}, cancelledIDs,
		"exactly the 9 runs newer than the oldest queued run should be cancelled")

	remaining := remainingConcurrencySlotTaskIDs(t, ctx, pool, childStrategyID)
	assert.ElementsMatch(t, []int64{100, 101}, remaining,
		"the running run (100) and the oldest queued run (101) must both still have a concurrency slot")
}

// TestCancelExceptOldestWithParentStrategy_PromotesSurvivorOnNextPoll mirrors
// TestCancelExceptNewestWithParentStrategy_PromotesSurvivorOnNextPoll: once the occupying run's slot
// frees, the surviving oldest queued run must be promoted on the very next poll.
func TestCancelExceptOldestWithParentStrategy_PromotesSurvivorOnNextPoll(t *testing.T) {
	t.Parallel()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	repo := createConcurrencyRepositoryForTest(pool)

	tenantID := uuid.New()
	now := time.Now().UTC()

	workflowID, workflowVersionID, parentStrategyID, childStrategyID := seedParentChildConcurrency(
		t, ctx, pool, tenantID, "CANCEL_EXCEPT_OLDEST", 1,
	)

	insertParentedConcurrencySlot(t, ctx, pool, tenantID, workflowID, workflowVersionID, parentStrategyID, childStrategyID, 200, now, true)
	survivorRunID := insertParentedConcurrencySlot(t, ctx, pool, tenantID, workflowID, workflowVersionID, parentStrategyID, childStrategyID, 201, now.Add(time.Second), false)
	insertParentedConcurrencySlot(t, ctx, pool, tenantID, workflowID, workflowVersionID, parentStrategyID, childStrategyID, 202, now.Add(2*time.Second), false)

	strategy := &sqlcv1.V1StepConcurrency{
		ID:                childStrategyID,
		ParentStrategyID:  pgtype.Int8{Int64: parentStrategyID, Valid: true},
		WorkflowID:        workflowID,
		WorkflowVersionID: workflowVersionID,
		Strategy:          sqlcv1.V1ConcurrencyStrategyCANCELEXCEPTOLDEST,
		TenantID:          tenantID,
		MaxConcurrency:    1,
	}

	// first poll: 202 should be cancelled, 201 (oldest queued) should survive queued, 200 keeps running.
	_, err := repo.RunConcurrencyStrategy(ctx, tenantID, strategy)
	require.NoError(t, err)

	remaining := remainingConcurrencySlotTaskIDs(t, ctx, pool, childStrategyID)
	require.ElementsMatch(t, []int64{200, 201}, remaining, "202 should already be cancelled after the first poll")

	// simulate task 200 completing: its concurrency slot and admission-gate row are released, exactly
	// as after_v1_concurrency_slot_delete_function() would do for a real task completion.
	_, err = pool.Exec(ctx, `DELETE FROM v1_concurrency_slot WHERE strategy_id = $1 AND task_id = 200`, childStrategyID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `DELETE FROM v1_workflow_concurrency_slot WHERE strategy_id = $1 AND workflow_run_id != $2`, parentStrategyID, survivorRunID)
	require.NoError(t, err)

	// second poll: the slot is free, so the survivor (201) must be promoted to running now - not on
	// some later poll, and without needing a new arrival on this key.
	res, err := repo.RunConcurrencyStrategy(ctx, tenantID, strategy)
	require.NoError(t, err)
	require.NotNil(t, res)

	queuedIDs := make([]int64, 0, len(res.Queued))
	for _, q := range res.Queued {
		queuedIDs = append(queuedIDs, q.Id)
	}

	assert.ElementsMatch(t, []int64{201}, queuedIDs, "the survivor must be promoted as soon as the slot frees")
}
