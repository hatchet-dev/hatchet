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

	result, err := ingestDagStepTriggerResult(ctx, repo, tenantID, task, invocationCount, actionId, childIndex)
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
					WorkflowName:         "my-dag",
					Data:                 []byte(`{"x":1}`),
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

// The operator replays steps in the order their completions were originally delivered, so a
// replay should never plan a step at a node another step already holds. If it does, the two
// steps share a workflow name and input, and only the step identity in the idempotency key
// stops the replay from silently resolving one step to the other's entry.
func TestDagStepReplayAtAnotherStepsNodeIsNondeterministic(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	repos := newUserEventScopeTestRepositories(t, pool)
	task := createUserEventScopeTestTask(t, ctx, repos, tenantID, 201)

	dedupe := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, task, 1, "my-dag:dedupe-action-items", 13)
	require.False(t, dedupe.AlreadyExisted)
	require.EqualValues(t, 1, dedupe.NodeId)

	reinvokeDurableTask(t, ctx, repos, tenantID, task)

	_, err := ingestDagStepTriggerResult(ctx, repos.durable, tenantID, task, 2, "my-dag:bold-dates-and-quantities", 9)

	var nonDeterminismErr *NonDeterminismError
	require.ErrorAs(t, err, &nonDeterminismErr)
	require.EqualValues(t, 1, nonDeterminismErr.NodeId)
}

// Two different steps of the same DAG carry the same workflow name and input; their entries
// must still be distinguishable, or a collision resolves one step to the other's child.
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
