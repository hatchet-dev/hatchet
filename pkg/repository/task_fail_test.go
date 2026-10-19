//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"fmt"
	"testing"

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
			Input:         &TaskInput{Input: map[string]any{"key": "value"}},
			StepIndex:     0,
			InitialState:  sqlcv1.V1TaskInitialStateQUEUED,
		},
	}, stepIdsToConfig)
	require.NoError(t, err)
	require.Len(t, insertedTasks, 1)

	return insertedTasks[0]
}

func TestFailTasksRetryDecisionIsPerTask(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := newBatchTestRepository(pool)

	ctx := context.Background()

	cases := []struct {
		name           string
		retries        int
		hasRuntime     bool
		isAppError     bool
		isNonRetryable bool
		wantRetry      bool
	}{
		{name: "app error, step allows no retries", retries: 0, hasRuntime: true, isAppError: true},
		{name: "app error, step allows retries", retries: 2, hasRuntime: true, isAppError: true, wantRetry: true},
		{name: "app error, non-retryable", retries: 2, hasRuntime: true, isAppError: true, isNonRetryable: true},
		{name: "app error, no runtime", retries: 2, isAppError: true},
		{name: "internal error", hasRuntime: true, wantRetry: true},
		{name: "internal error, no runtime"},
	}

	names := make(map[int64]string, len(cases))
	opts := make([]FailTaskOpts, 0, len(cases))
	runtimes := sqlcv1.InsertBufferedTaskRuntimesParams{Tenantid: internalTenantId}
	want := make([]string, 0, len(cases))

	for i, c := range cases {
		task := insertTaskWithRetries(t, ctx, repo, fmt.Sprintf("fail-batch-%d", i), c.retries)
		names[task.ID] = c.name

		opts = append(opts, FailTaskOpts{
			TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				RetryCount: task.RetryCount,
			},
			IsAppError:     c.isAppError,
			IsNonRetryable: c.isNonRetryable,
			ErrorMessage:   "task failed",
		})

		if c.hasRuntime {
			runtimes.Taskids = append(runtimes.Taskids, task.ID)
			runtimes.Taskinsertedats = append(runtimes.Taskinsertedats, task.InsertedAt)
			runtimes.Taskretrycounts = append(runtimes.Taskretrycounts, task.RetryCount)
		}

		if c.wantRetry {
			want = append(want, c.name)
		}
	}

	require.NoError(t, repo.queries.InsertBufferedTaskRuntimes(ctx, repo.pool, runtimes))

	res, err := repo.FailTasks(ctx, internalTenantId, opts)
	require.NoError(t, err)

	got := make([]string, 0, len(res.RetriedTasks))
	for _, r := range res.RetriedTasks {
		got = append(got, names[r.Id])
	}

	require.ElementsMatch(t, want, got)
}
