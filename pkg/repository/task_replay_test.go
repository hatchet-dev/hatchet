//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// replaying a task that already has a queue item used to create a duplicate queue item for the same
// task at a different retry count. the preflight check must reject replays of tasks that are still
// waiting in the queue so this can't happen.
func TestReplayQueuedTaskIsRejected(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := newBatchTestRepository(pool)
	workflows := &workflowRepository{sharedRepository: repo.sharedRepository}

	ctx := context.Background()

	wf, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("replay-queued-task", "v1", nil))
	require.NoError(t, err)

	steps, err := repo.queries.ListStepsByWorkflowVersionIds(ctx, pool, sqlcv1.ListStepsByWorkflowVersionIdsParams{
		Ids:      []uuid.UUID{wf.WorkflowVersion.ID},
		Tenantid: internalTenantId,
	})
	require.NoError(t, err)
	require.Len(t, steps, 1)

	stepID := steps[0].ID

	stepIdsToConfig, err := repo.sharedRepository.listStepsByIds(ctx, pool, internalTenantId, []uuid.UUID{stepID})
	require.NoError(t, err)
	require.Contains(t, stepIdsToConfig, stepID)

	insertedTasks, err := repo.sharedRepository.insertTasks(ctx, repo.pool, internalTenantId, []CreateTaskOpts{
		{
			ExternalId:    uuid.New(),
			WorkflowRunId: uuid.New(),
			StepId:        stepID,
			Input:         &TaskInput{Input: map[string]interface{}{"key": "value"}},
			StepIndex:     0,
			InitialState:  sqlcv1.V1TaskInitialStateQUEUED,
		},
	}, stepIdsToConfig)
	require.NoError(t, err)
	require.Len(t, insertedTasks, 1)

	task := insertedTasks[0]

	listQueueItemRetryCounts := func() []int32 {
		rows, err := pool.Query(ctx, `
			SELECT retry_count FROM v1_queue_item WHERE task_id = $1 ORDER BY retry_count`,
			task.ID,
		)
		require.NoError(t, err)
		defer rows.Close()

		var retryCounts []int32
		for rows.Next() {
			var retryCount int32
			require.NoError(t, rows.Scan(&retryCount))
			retryCounts = append(retryCounts, retryCount)
		}
		require.NoError(t, rows.Err())
		return retryCounts
	}

	require.Equal(t, []int32{0}, listQueueItemRetryCounts())

	replayed, err := repo.ReplayTasks(ctx, internalTenantId, []TaskIdInsertedAtRetryCount{
		{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			RetryCount: task.RetryCount,
		},
	})
	require.NoError(t, err)
	require.Empty(t, replayed.ReplayedTasks,
		"replaying a task that is still waiting in the queue must be rejected by the preflight check")

	var taskRetryCount int32
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT retry_count FROM v1_task WHERE id = $1`, task.ID).Scan(&taskRetryCount))
	require.Equal(t, int32(0), taskRetryCount)

	require.Equal(t, []int32{0}, listQueueItemRetryCounts(),
		"a rejected replay must leave the existing queue item untouched")
}

// with the dag operator enabled the replay path relies on row locks instead of the tenant-wide
// advisory lock, so two replays of the same task racing for the row lock must both succeed and
// bump the retry count exactly once.
func TestConcurrentReplaysWithDagOperatorEnabledIncrementRetryCountOnce(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := newBatchTestRepository(pool)
	shared := repo.sharedRepository
	workflows := &workflowRepository{sharedRepository: shared}

	ctx := context.Background()

	_, err := repo.queries.UpsertTenantEntitlement(ctx, pool, sqlcv1.UpsertTenantEntitlementParams{
		Tenantid:    internalTenantId,
		Dagoperator: true,
	})
	require.NoError(t, err)

	workflowOpts := minimalWorkflowOpts("replay-concurrent-dag-operator", "v1", nil)

	_, err = workflows.PutWorkflowVersion(ctx, internalTenantId, workflowOpts)
	require.NoError(t, err)

	tasks, _, _, _, err := newTriggerRepository(shared).TriggerFromWorkflowNames(ctx, internalTenantId, []*WorkflowNameTriggerOpts{
		{
			TriggerTaskData: &TriggerTaskData{
				WorkflowName: workflowOpts.Name,
				Data:         []byte(`{}`),
			},
			ExternalId: uuid.New(),
		},
	})
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	task := tasks[0]

	_, err = repo.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{
		{
			TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				RetryCount: task.RetryCount,
			},
			Output: []byte(`{}`),
		},
	})
	require.NoError(t, err)

	replayOpts := []TaskIdInsertedAtRetryCount{
		{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			RetryCount: task.RetryCount,
		},
	}

	// hold the task's row lock so both replays block on it, then release it so they race from the
	// same starting point
	rowLockHolder, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer rowLockHolder.Rollback(ctx) //nolint:errcheck

	_, err = repo.queries.ListTasksForReplay(ctx, rowLockHolder, sqlcv1.ListTasksForReplayParams{
		Taskids:           []int64{task.ID},
		Taskinsertedats:   []pgtype.Timestamptz{task.InsertedAt},
		Mintaskinsertedat: task.InsertedAt,
		Tenantid:          internalTenantId,
	})
	require.NoError(t, err)

	errs := make(chan error, 2)

	for i := 0; i < 2; i++ {
		go func() {
			_, err := repo.ReplayTasks(ctx, internalTenantId, replayOpts)
			errs <- err
		}()
	}

	require.Eventually(t, func() bool {
		var blockedBackends int
		err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM pg_stat_activity WHERE wait_event_type = 'Lock'`).Scan(&blockedBackends)
		return err == nil && blockedBackends == 2
	}, 10*time.Second, 10*time.Millisecond, "both replays should be waiting on the row lock")

	require.NoError(t, rowLockHolder.Rollback(ctx))

	for i := 0; i < 2; i++ {
		require.NoError(t, <-errs)
	}

	flattened, err := repo.FlattenExternalIds(ctx, internalTenantId, []uuid.UUID{task.ExternalID})
	require.NoError(t, err)
	require.Len(t, flattened, 1)
	require.Equal(t, task.RetryCount+1, flattened[0].RetryCount)
}
