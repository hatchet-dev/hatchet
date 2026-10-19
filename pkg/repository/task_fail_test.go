//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func insertTaskWithRetries(t *testing.T, ctx context.Context, repo *TaskRepositoryImpl, workflowName string, retries int) *V1TaskWithPayload {
	t.Helper()

	workflows := &workflowRepository{sharedRepository: repo.sharedRepository}

	opts := minimalWorkflowOpts(workflowName, "v1", nil)
	opts.Tasks[0].Retries = &retries

	wf, err := workflows.PutWorkflowVersion(ctx, internalTenantId, opts)
	require.NoError(t, err)

	steps, err := repo.queries.ListStepsByWorkflowVersionIds(ctx, repo.pool, sqlcv1.ListStepsByWorkflowVersionIdsParams{
		Ids:      []uuid.UUID{wf.WorkflowVersion.ID},
		Tenantid: internalTenantId,
	})
	require.NoError(t, err)
	require.Len(t, steps, 1)

	stepID := steps[0].ID

	stepIdsToConfig, err := repo.sharedRepository.listStepsByIds(ctx, repo.pool, internalTenantId, []uuid.UUID{stepID})
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

	return insertedTasks[0]
}

func startTaskRuntime(t *testing.T, ctx context.Context, repo *TaskRepositoryImpl, task *V1TaskWithPayload) {
	t.Helper()

	_, err := repo.pool.Exec(ctx, `
		INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, tenant_id, timeout_at)
		VALUES ($1, $2, $3, $4, $5)
	`, task.ID, task.InsertedAt, task.RetryCount, internalTenantId, time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
}

type taskRetryCounts struct {
	retryCount         int32
	appRetryCount      int32
	internalRetryCount int32
}

func getTaskRetryCounts(t *testing.T, ctx context.Context, repo *TaskRepositoryImpl, task *V1TaskWithPayload) taskRetryCounts {
	t.Helper()

	var counts taskRetryCounts

	require.NoError(t, repo.pool.QueryRow(ctx, `
		SELECT retry_count, app_retry_count, internal_retry_count
		FROM v1_task
		WHERE id = $1 AND inserted_at = $2`,
		task.ID, task.InsertedAt,
	).Scan(&counts.retryCount, &counts.appRetryCount, &counts.internalRetryCount))

	return counts
}

func failTaskOpts(task *V1TaskWithPayload, isAppError bool) FailTaskOpts {
	return FailTaskOpts{
		TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			RetryCount: task.RetryCount,
		},
		IsAppError:   isAppError,
		ErrorMessage: "task failed",
	}
}

func retriedTaskIds(res *FailTasksResponse) []int64 {
	ids := make([]int64, 0, len(res.RetriedTasks))

	for _, task := range res.RetriedTasks {
		ids = append(ids, task.Id)
	}

	return ids
}

func TestFailTasksRetryDecisionIsPerTask(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := newBatchTestRepository(pool)

	ctx := context.Background()

	t.Run("app failure uses each task's own step retries", func(t *testing.T) {
		noRetries := insertTaskWithRetries(t, ctx, repo, "fail-batch-no-retries", 0)
		withRetries := insertTaskWithRetries(t, ctx, repo, "fail-batch-with-retries", 2)

		startTaskRuntime(t, ctx, repo, noRetries)
		startTaskRuntime(t, ctx, repo, withRetries)

		res, err := repo.FailTasks(ctx, internalTenantId, []FailTaskOpts{
			failTaskOpts(noRetries, true),
			failTaskOpts(withRetries, true),
		})
		require.NoError(t, err)

		require.ElementsMatch(t, []int64{withRetries.ID}, retriedTaskIds(res))

		require.Equal(t, taskRetryCounts{}, getTaskRetryCounts(t, ctx, repo, noRetries),
			"a task whose step allows no retries must not be retried because another task in the batch allows retries")
		require.Equal(t, taskRetryCounts{retryCount: 1, appRetryCount: 1}, getTaskRetryCounts(t, ctx, repo, withRetries))
	})

	t.Run("app failure retries only tasks that are not marked non-retryable", func(t *testing.T) {
		retryable := insertTaskWithRetries(t, ctx, repo, "fail-batch-retryable", 2)
		nonRetryable := insertTaskWithRetries(t, ctx, repo, "fail-batch-non-retryable", 2)

		startTaskRuntime(t, ctx, repo, retryable)
		startTaskRuntime(t, ctx, repo, nonRetryable)

		nonRetryableOpts := failTaskOpts(nonRetryable, true)
		nonRetryableOpts.IsNonRetryable = true

		res, err := repo.FailTasks(ctx, internalTenantId, []FailTaskOpts{
			failTaskOpts(retryable, true),
			nonRetryableOpts,
		})
		require.NoError(t, err)

		require.ElementsMatch(t, []int64{retryable.ID}, retriedTaskIds(res))

		require.Equal(t, taskRetryCounts{retryCount: 1, appRetryCount: 1}, getTaskRetryCounts(t, ctx, repo, retryable))
		require.Equal(t, taskRetryCounts{}, getTaskRetryCounts(t, ctx, repo, nonRetryable),
			"a non-retryable task must not be retried because another task in the batch is retryable")
	})

	t.Run("app failure skips tasks without a runtime", func(t *testing.T) {
		running := insertTaskWithRetries(t, ctx, repo, "fail-batch-app-running", 2)
		released := insertTaskWithRetries(t, ctx, repo, "fail-batch-app-released", 2)

		startTaskRuntime(t, ctx, repo, running)

		res, err := repo.FailTasks(ctx, internalTenantId, []FailTaskOpts{
			failTaskOpts(running, true),
			failTaskOpts(released, true),
		})
		require.NoError(t, err)

		require.ElementsMatch(t, []int64{running.ID}, retriedTaskIds(res))

		require.Equal(t, taskRetryCounts{retryCount: 1, appRetryCount: 1}, getTaskRetryCounts(t, ctx, repo, running))
		require.Equal(t, taskRetryCounts{}, getTaskRetryCounts(t, ctx, repo, released),
			"a late failure for an attempt without a runtime must not be retried because another task in the batch has one")
	})

	t.Run("internal failure skips tasks without a runtime", func(t *testing.T) {
		running := insertTaskWithRetries(t, ctx, repo, "fail-batch-internal-running", 0)
		released := insertTaskWithRetries(t, ctx, repo, "fail-batch-internal-released", 0)

		startTaskRuntime(t, ctx, repo, running)

		res, err := repo.FailTasks(ctx, internalTenantId, []FailTaskOpts{
			failTaskOpts(running, false),
			failTaskOpts(released, false),
		})
		require.NoError(t, err)

		require.ElementsMatch(t, []int64{running.ID}, retriedTaskIds(res))

		require.Equal(t, taskRetryCounts{retryCount: 1, internalRetryCount: 1}, getTaskRetryCounts(t, ctx, repo, running))
		require.Equal(t, taskRetryCounts{}, getTaskRetryCounts(t, ctx, repo, released),
			"a late internal failure for an attempt without a runtime must not be retried because another task in the batch has one")
	})
}
