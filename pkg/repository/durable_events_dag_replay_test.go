//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ingestDagStepTrigger(
	t *testing.T,
	ctx context.Context,
	repo *durableEventsRepository,
	tenantID uuid.UUID,
	task *sqlcv1.FlattenExternalIdsRow,
	invocationCount int32,
	actionId string,
	childIndex int64,
) *IngestTriggerRunsEntry {
	t.Helper()

	result, err := ingestDagStepTriggerResult(ctx, repo, tenantID, task, invocationCount, actionId, childIndex, []byte(`{"x":1}`), false)
	require.NoError(t, err)
	require.NotNil(t, result.TriggerRunsResult)
	require.Len(t, result.TriggerRunsResult.Entries, 1)

	return result.TriggerRunsResult.Entries[0]
}

func ingestDagStepTriggerResult(
	ctx context.Context,
	repo *durableEventsRepository,
	tenantID uuid.UUID,
	task *sqlcv1.FlattenExternalIdsRow,
	invocationCount int32,
	actionId string,
	childIndex int64,
	input []byte,
	isSkipped bool,
) (*IngestDurableTaskEventResult, error) {
	parentExternalId := task.ExternalID
	parentTaskId := task.ID
	parentTaskInsertedAt := task.InsertedAt.Time

	return repo.IngestDurableTaskEvent(ctx, IngestDurableTaskEventOpts{
		BaseIngestEventOpts: &BaseIngestEventOpts{
			Task:            task,
			Kind:            sqlcv1.V1DurableEventLogKindRUN,
			InvocationCount: invocationCount,
			TenantId:        tenantID,
		},
		TriggerRuns: &IngestTriggerRunsOpts{
			TriggerOpts: []*WorkflowNameTriggerOpts{{
				IsDagStepTrigger:       true,
				ReplayOrphanedChildren: true,
				TriggerTaskData: &TriggerTaskData{
					IsSkipped:            isSkipped,
					WorkflowName:         "my-dag",
					Data:                 input,
					TargetActionId:       &actionId,
					ParentExternalId:     &parentExternalId,
					ParentTaskId:         &parentTaskId,
					ParentTaskInsertedAt: &parentTaskInsertedAt,
					ChildIndex:           &childIndex,
				},
			}},
		},
	})
}

func reinvokeDurableTask(t *testing.T, ctx context.Context, repos userEventScopeTestRepositories, tenantID uuid.UUID, task *sqlcv1.FlattenExternalIdsRow) {
	t.Helper()

	_, err := repos.shared.queries.IncrementLogFileInvocationCounts(ctx, repos.shared.pool, sqlcv1.IncrementLogFileInvocationCountsParams{
		Durabletaskids:         []int64{task.ID},
		Durabletaskinsertedats: []pgtype.Timestamptz{task.InsertedAt},
		Tenantids:              []uuid.UUID{tenantID},
	})
	require.NoError(t, err)
}

func TestDagStepReplayReusesTheEntryEvenWhenTheInputBytesDiffer(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	repos := newUserEventScopeTestRepositories(t, pool)
	task := createUserEventScopeTestTask(t, ctx, repos, tenantID, 201)

	first := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, task, 1, "my-dag:step-a", 1)
	require.False(t, first.AlreadyExisted)
	require.EqualValues(t, 1, first.NodeId)

	reinvokeDurableTask(t, ctx, repos, tenantID, task)

	result, err := ingestDagStepTriggerResult(ctx, repos.durable, tenantID, task, 2, "my-dag:step-a", 1, []byte(`{"x": 1}`), false)
	require.NoError(t, err)
	require.Len(t, result.TriggerRunsResult.Entries, 1)

	replayed := result.TriggerRunsResult.Entries[0]
	require.True(t, replayed.AlreadyExisted)
	require.EqualValues(t, 1, replayed.NodeId)
	require.Equal(t, first.WorkflowRunExternalId, replayed.WorkflowRunExternalId)
}

func TestDagStepEntriesOfSameWorkflowHaveDistinctIdempotencyKeys(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	repos := newUserEventScopeTestRepositories(t, pool)
	task := createUserEventScopeTestTask(t, ctx, repos, tenantID, 202)

	first := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, task, 1, "my-dag:step-a", 1)
	second := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, task, 1, "my-dag:step-b", 2)

	entries, err := repos.shared.queries.GetDurableEventLogEntriesByChildTaskExternalIds(ctx, repos.shared.pool, sqlcv1.GetDurableEventLogEntriesByChildTaskExternalIdsParams{
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Childtaskexternalids:  []uuid.UUID{first.WorkflowRunExternalId, second.WorkflowRunExternalId},
	})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.NotEqual(t, entries[0].IdempotencyKey, entries[1].IdempotencyKey)
}

func TestDagStepCreatedSkippedTakesNextSatisfiedOrder(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	repos := newUserEventScopeTestRepositories(t, pool)
	task := createUserEventScopeTestTask(t, ctx, repos, tenantID, 203)

	running := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, task, 1, "my-dag:step-a", 1)
	require.False(t, running.IsSatisfied)
	require.Nil(t, running.SatisfiedOrder)

	result, err := ingestDagStepTriggerResult(ctx, repos.durable, tenantID, task, 1, "my-dag:step-b", 2, []byte(`{"x":1}`), true)
	require.NoError(t, err)

	skipped := result.TriggerRunsResult.Entries[0]
	require.True(t, skipped.IsSatisfied)
	require.NotNil(t, skipped.SatisfiedOrder)
	require.EqualValues(t, 1, *skipped.SatisfiedOrder)

	logFile, err := repos.shared.queries.GetDurableTaskLogFiles(ctx, repos.shared.pool, sqlcv1.GetDurableTaskLogFilesParams{
		Durabletaskids:         []int64{task.ID},
		Durabletaskinsertedats: []pgtype.Timestamptz{task.InsertedAt},
		Tenantids:              []uuid.UUID{tenantID},
	})
	require.NoError(t, err)
	require.Len(t, logFile, 1)
	require.EqualValues(t, 1, logFile[0].LatestSatisfiedOrder)
}
