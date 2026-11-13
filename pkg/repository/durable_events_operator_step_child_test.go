//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func releasedOperatorStepChild(orchestrator *sqlcv1.FlattenExternalIdsRow, childTaskId int64, childExternalId uuid.UUID) *sqlcv1.ReleaseTasksRow {
	return &sqlcv1.ReleaseTasksRow{
		ID:                   childTaskId,
		InsertedAt:           pgtype.Timestamptz{Time: orchestrator.InsertedAt.Time, Valid: true},
		ExternalID:           childExternalId,
		WorkflowRunID:        orchestrator.ExternalID,
		ParentTaskExternalID: &orchestrator.ExternalID,
		ParentTaskID:         pgtype.Int8{Int64: orchestrator.ID, Valid: true},
		ParentTaskInsertedAt: orchestrator.InsertedAt,
		IsCurrentRetry:       true,
	}
}

func satisfyOperatorStepChildren(t *testing.T, ctx context.Context, repos userEventScopeTestRepositories, tenantID uuid.UUID, completions []OperatorStepChildCompletion) []SatisfiedEntry {
	t.Helper()

	tx, err := repos.shared.pool.ForShared().Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) // nolint: errcheck

	entries, err := repos.shared.satisfyOperatorStepChildEntries(ctx, tx, tenantID, completions)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	return entries
}

func TestIsOperatorStepChildRequiresTheOrchestratorAsParentAndRun(t *testing.T) {
	orchestratorExternalId := uuid.New()
	otherExternalId := uuid.New()

	operatorChild := &sqlcv1.ReleaseTasksRow{
		WorkflowRunID:        orchestratorExternalId,
		ParentTaskExternalID: &orchestratorExternalId,
		ParentTaskID:         pgtype.Int8{Int64: 1, Valid: true},
		ParentTaskInsertedAt: pgtype.Timestamptz{Valid: true},
	}
	require.True(t, IsOperatorStepChild(operatorChild))

	regularChildWorkflow := &sqlcv1.ReleaseTasksRow{
		WorkflowRunID:        otherExternalId,
		ParentTaskExternalID: &orchestratorExternalId,
		ParentTaskID:         pgtype.Int8{Int64: 1, Valid: true},
		ParentTaskInsertedAt: pgtype.Timestamptz{Valid: true},
	}
	require.False(t, IsOperatorStepChild(regularChildWorkflow))

	rootTask := &sqlcv1.ReleaseTasksRow{WorkflowRunID: otherExternalId}
	require.False(t, IsOperatorStepChild(rootTask))
}

func TestOperatorStepChildCompletionSatisfiesItsEntryWithoutAMatch(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	repos := newUserEventScopeTestRepositories(t, pool)
	orchestrator := createUserEventScopeTestTask(t, ctx, repos, tenantID, 301)

	stepA := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, orchestrator, 1, "my-dag:step-a", 1)
	stepB := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, orchestrator, 1, "my-dag:step-b", 2)
	require.False(t, stepA.IsSatisfied)
	require.False(t, stepB.IsSatisfied)

	childA := releasedOperatorStepChild(orchestrator, 1001, stepA.WorkflowRunExternalId)
	childB := releasedOperatorStepChild(orchestrator, 1002, stepB.WorkflowRunExternalId)

	entries := satisfyOperatorStepChildren(t, ctx, repos, tenantID, []OperatorStepChildCompletion{
		{ReleasedTask: childA, OutputEvent: &TaskOutputEvent{EventType: sqlcv1.V1TaskEventTypeCOMPLETED, Output: []byte(`{"answer":42}`)}},
		{ReleasedTask: childB, OutputEvent: &TaskOutputEvent{EventType: sqlcv1.V1TaskEventTypeFAILED, IsFailure: true, ErrorMessage: "boom"}},
	})
	require.Len(t, entries, 2)

	entriesByNode := make(map[int64]SatisfiedEntry, len(entries))
	for _, entry := range entries {
		entriesByNode[entry.NodeId] = entry
	}

	completed := entriesByNode[stepA.NodeId]
	require.Equal(t, orchestrator.ExternalID, completed.DurableTaskExternalId)
	require.False(t, completed.ChildTaskIsFailure)
	require.JSONEq(t, `{"answer":42}`, string(completed.Data))
	require.NotNil(t, completed.SatisfiedOrder)

	failed := entriesByNode[stepB.NodeId]
	require.True(t, failed.ChildTaskIsFailure)
	require.NotNil(t, failed.ChildTaskErrorMessage)
	require.Equal(t, "boom", *failed.ChildTaskErrorMessage)
	require.NotNil(t, failed.SatisfiedOrder)
	require.NotEqual(t, *completed.SatisfiedOrder, *failed.SatisfiedOrder)

	storedEntries, err := repos.shared.queries.BulkGetDurableEventLogEntries(ctx, repos.shared.pool.ForShared(), sqlcv1.BulkGetDurableEventLogEntriesParams{
		Durabletaskids:         []int64{orchestrator.ID, orchestrator.ID},
		Durabletaskinsertedats: []pgtype.Timestamptz{orchestrator.InsertedAt, orchestrator.InsertedAt},
		Branchids:              []int64{stepA.BranchId, stepB.BranchId},
		Nodeids:                []int64{stepA.NodeId, stepB.NodeId},
	})
	require.NoError(t, err)
	require.Len(t, storedEntries, 2)

	for _, entry := range storedEntries {
		require.True(t, entry.IsSatisfied)

		switch entry.NodeID {
		case stepA.NodeId:
			require.False(t, entry.ChildTaskIsFailure)

			result, err := repos.shared.payloadStore.DurableEventLogEntryResultData().RetrieveOne(ctx, repos.shared.pool.ForShared(), PayloadRef[DurableEventLogEntryResultDataShape]{
				Id:         entry.ID,
				InsertedAt: entry.InsertedAt,
				ExternalId: entry.ResultPayloadExternalID,
				TenantId:   tenantID,
			})
			require.NoError(t, err)
			require.JSONEq(t, `{"answer":42}`, string(result.Bytes()))
		case stepB.NodeId:
			require.True(t, entry.ChildTaskIsFailure)
			require.Equal(t, "boom", entry.ChildTaskErrorMessage.String)
		}
	}

	// a duplicate completion report finds nothing left to satisfy
	again := satisfyOperatorStepChildren(t, ctx, repos, tenantID, []OperatorStepChildCompletion{
		{ReleasedTask: childA, OutputEvent: &TaskOutputEvent{EventType: sqlcv1.V1TaskEventTypeCOMPLETED, Output: []byte(`{"answer":43}`)}},
	})
	require.Empty(t, again)

	conditions, err := repos.shared.queries.ListMatchConditionsForEventWithHint(ctx, repos.shared.pool.ForShared(), sqlcv1.ListMatchConditionsForEventWithHintParams{
		Tenantid:           tenantID,
		Eventtype:          sqlcv1.V1EventTypeINTERNAL,
		Eventkeys:          []string{string(sqlcv1.V1TaskEventTypeCOMPLETED), string(sqlcv1.V1TaskEventTypeCOMPLETED)},
		Eventresourcehints: []string{stepA.WorkflowRunExternalId.String(), stepB.WorkflowRunExternalId.String()},
	})
	require.NoError(t, err)
	require.Empty(t, conditions)
}

func TestOperatorStepChildCancellationSatisfiesTheEntryAsAFailure(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	repos := newUserEventScopeTestRepositories(t, pool)
	orchestrator := createUserEventScopeTestTask(t, ctx, repos, tenantID, 302)

	step := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, orchestrator, 1, "my-dag:step-a", 1)

	entries := satisfyOperatorStepChildren(t, ctx, repos, tenantID, []OperatorStepChildCompletion{{
		ReleasedTask: releasedOperatorStepChild(orchestrator, 1003, step.WorkflowRunExternalId),
		OutputEvent:  &TaskOutputEvent{EventType: sqlcv1.V1TaskEventTypeCANCELLED},
	}})
	require.Len(t, entries, 1)
	require.True(t, entries[0].ChildTaskIsFailure)
	require.NotNil(t, entries[0].ChildTaskErrorMessage)
	require.Equal(t, TaskCancelledErrorMessage, *entries[0].ChildTaskErrorMessage)
}

const operatorStepChildMatchTestAction = "integration:step1"

// newOperatorStepChildMatchTestTasks registers the workflow ingestDagStepTrigger names and turns
// direct completion off for the tenant, so spawning a step child writes a completion match.
func newOperatorStepChildMatchTestTasks(t *testing.T, ctx context.Context, pool *pgxpool.Pool, repos userEventScopeTestRepositories) *TaskRepositoryImpl {
	t.Helper()

	require.NoError(t, newTenantEntitlementRepository(repos.shared).SetEntitlements(ctx, internalTenantId, TenantEntitlements{OperatorStepDirectCompletion: false}))

	_, err := (&workflowRepository{sharedRepository: repos.shared}).PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("my-dag", "v1", nil))
	require.NoError(t, err)

	return newBatchTestRepository(pool)
}

// spawnOperatorStepChild ingests a step trigger and creates its child task, as the dispatcher does.
func spawnOperatorStepChild(t *testing.T, ctx context.Context, repos userEventScopeTestRepositories, orchestrator *sqlcv1.FlattenExternalIdsRow, childIndex int64) (*IngestTriggerRunsEntry, *TaskIdInsertedAtRetryCount) {
	t.Helper()

	result, err := ingestDagStepTriggerResult(ctx, repos.durable, internalTenantId, orchestrator, 1, operatorStepChildMatchTestAction, childIndex, []byte(`{}`), false)
	require.NoError(t, err)

	created, _, _, err := repos.durable.TriggerPendingRunEntries(ctx, internalTenantId, []TriggerPendingRunEntriesOpt{{
		Task:        orchestrator,
		PendingRuns: result.TriggerRunsResult.PendingTriggers,
	}})
	require.NoError(t, err)
	require.Len(t, created, 1)

	return result.TriggerRunsResult.Entries[0], &TaskIdInsertedAtRetryCount{Id: created[0].ID, InsertedAt: created[0].InsertedAt, RetryCount: created[0].RetryCount}
}

// countMatchRows counts every match and match condition. Each test database holds one
// orchestrator, so they all wait on its step children.
func countMatchRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (rows int) {
	t.Helper()

	require.NoError(t, pool.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM v1_match) + (SELECT COUNT(*) FROM v1_match_condition)`).Scan(&rows))

	return rows
}

func TestOperatorStepChildFinalizeDeletesItsCompletionMatch(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repos := newUserEventScopeTestRepositories(t, pool)
	tasks := newOperatorStepChildMatchTestTasks(t, ctx, pool, repos)
	orchestrator := createUserEventScopeTestTask(t, ctx, repos, internalTenantId, 303)

	_, completes := spawnOperatorStepChild(t, ctx, repos, orchestrator, 1)
	_, fails := spawnOperatorStepChild(t, ctx, repos, orchestrator, 2)
	_, cancels := spawnOperatorStepChild(t, ctx, repos, orchestrator, 3)
	require.Equal(t, 12, countMatchRows(t, ctx, pool), "with direct completion off, spawning a step child writes a match and three conditions")

	completed, err := tasks.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{{TaskIdInsertedAtRetryCount: completes, Output: []byte(`{"answer":42}`)}})
	require.NoError(t, err)
	require.Len(t, completed.SatisfiedDurableEventLogEntries, 1)
	require.Equal(t, 8, countMatchRows(t, ctx, pool))

	failed, err := tasks.FailTasks(ctx, internalTenantId, []FailTaskOpts{{TaskIdInsertedAtRetryCount: fails, IsAppError: true, ErrorMessage: "boom", IsNonRetryable: true}})
	require.NoError(t, err)
	require.Len(t, failed.SatisfiedDurableEventLogEntries, 1)
	require.Equal(t, 4, countMatchRows(t, ctx, pool))

	cancelled, err := tasks.CancelTasks(ctx, internalTenantId, []TaskIdInsertedAtRetryCount{*cancels})
	require.NoError(t, err)
	require.Len(t, cancelled.SatisfiedDurableEventLogEntries, 1)
	require.Zero(t, countMatchRows(t, ctx, pool))
}

func TestReExecutedOperatorStepChildFinalizeDeletesTheNewBranchMatch(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repos := newUserEventScopeTestRepositories(t, pool)
	tasks := newOperatorStepChildMatchTestTasks(t, ctx, pool, repos)
	orchestrator := createUserEventScopeTestTask(t, ctx, repos, internalTenantId, 304)

	firstEntry, child := spawnOperatorStepChild(t, ctx, repos, orchestrator, 1)

	_, err := tasks.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{{TaskIdInsertedAtRetryCount: child, Output: []byte(`{"run":1}`)}})
	require.NoError(t, err)
	require.Zero(t, countMatchRows(t, ctx, pool))

	_, err = repos.durable.HandleBranchForDAGReplay(ctx, internalTenantId, orchestrator, []uuid.UUID{firstEntry.WorkflowRunExternalId})
	require.NoError(t, err)
	reinvokeDurableTask(t, ctx, repos, internalTenantId, orchestrator)

	rerunEntry := ingestDagStepTrigger(t, ctx, repos.durable, internalTenantId, orchestrator, 2, operatorStepChildMatchTestAction, 1)
	require.True(t, rerunEntry.ChildNeedsReplay, "a re-executed child keeps its task")
	require.Equal(t, firstEntry.WorkflowRunExternalId, rerunEntry.WorkflowRunExternalId)
	require.NotEqual(t, firstEntry.BranchId, rerunEntry.BranchId)
	require.Equal(t, 4, countMatchRows(t, ctx, pool), "the new branch entry gets its own completion match")

	_, err = tasks.ReplayTasks(ctx, internalTenantId, []TaskIdInsertedAtRetryCount{*child})
	require.NoError(t, err)
	require.Equal(t, 8, countMatchRows(t, ctx, pool), "replaying the task adds a match that signals its parent")

	child.RetryCount++
	completed, err := tasks.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{{TaskIdInsertedAtRetryCount: child, Output: []byte(`{"run":2}`)}})
	require.NoError(t, err)
	require.Len(t, completed.SatisfiedDurableEventLogEntries, 1)
	require.Equal(t, rerunEntry.BranchId, completed.SatisfiedDurableEventLogEntries[0].BranchId)
	require.Zero(t, countMatchRows(t, ctx, pool))
}
