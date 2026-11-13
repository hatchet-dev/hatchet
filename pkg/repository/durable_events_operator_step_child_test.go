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

const operatorStepChildTestAction = "integration:step1"

func setOperatorStepDirectCompletion(t *testing.T, ctx context.Context, repos userEventScopeTestRepositories, isEntitled bool) {
	t.Helper()

	require.NoError(t, repos.shared.entitlements.SetEntitlements(ctx, internalTenantId, TenantEntitlements{OperatorStepDirectCompletion: isEntitled}))
}

func newOperatorStepChildTestTasks(t *testing.T, ctx context.Context, repos userEventScopeTestRepositories, step1Concurrency []CreateConcurrencyOpts) *TaskRepositoryImpl {
	t.Helper()

	workflowOpts := minimalWorkflowOpts("my-dag", "v1", nil)
	workflowOpts.Tasks[0].Concurrency = step1Concurrency

	_, err := (&workflowRepository{sharedRepository: repos.shared}).PutWorkflowVersion(ctx, internalTenantId, workflowOpts)
	require.NoError(t, err)

	return &TaskRepositoryImpl{
		sharedRepository:      repos.shared,
		taskRetentionPeriod:   24 * time.Hour,
		maxInternalRetryCount: 3,
	}
}

func spawnOperatorStepChild(t *testing.T, ctx context.Context, repos userEventScopeTestRepositories, orchestrator *sqlcv1.FlattenExternalIdsRow, childIndex int64) (*IngestTriggerRunsEntry, *V1TaskWithPayload) {
	t.Helper()

	result, err := ingestDagStepTriggerResult(ctx, repos.durable, internalTenantId, orchestrator, 1, operatorStepChildTestAction, childIndex, []byte(`{}`), false)
	require.NoError(t, err)

	created, _, _, err := repos.durable.TriggerPendingRunEntries(ctx, internalTenantId, []TriggerPendingRunEntriesOpt{{
		Task:        orchestrator,
		PendingRuns: result.TriggerRunsResult.PendingTriggers,
	}})
	require.NoError(t, err)
	require.Len(t, created, 1)

	return result.TriggerRunsResult.Entries[0], created[0]
}

func reExecuteOperatorStepChildOnNewBranch(t *testing.T, ctx context.Context, repos userEventScopeTestRepositories, orchestrator *sqlcv1.FlattenExternalIdsRow, firstEntry *IngestTriggerRunsEntry) *IngestTriggerRunsEntry {
	t.Helper()

	_, err := repos.durable.HandleBranchForDAGReplay(ctx, internalTenantId, orchestrator, []uuid.UUID{firstEntry.WorkflowRunExternalId})
	require.NoError(t, err)
	reinvokeDurableTask(t, ctx, repos, internalTenantId, orchestrator)

	rerunEntry := ingestDagStepTrigger(t, ctx, repos.durable, internalTenantId, orchestrator, 2, operatorStepChildTestAction, 1)
	require.True(t, rerunEntry.ChildNeedsReplay)
	require.NotEqual(t, firstEntry.BranchId, rerunEntry.BranchId)

	return rerunEntry
}

func listUnsatisfiedConditionsWaitingOnChild(t *testing.T, ctx context.Context, repos userEventScopeTestRepositories, childExternalId uuid.UUID) []*sqlcv1.ListMatchConditionsForEventRow {
	t.Helper()

	hint := childExternalId.String()

	conditions, err := repos.shared.queries.ListMatchConditionsForEventWithHint(ctx, repos.shared.pool.ForShared(), sqlcv1.ListMatchConditionsForEventWithHintParams{
		Tenantid:  internalTenantId,
		Eventtype: sqlcv1.V1EventTypeINTERNAL,
		Eventkeys: []string{
			string(sqlcv1.V1TaskEventTypeCOMPLETED),
			string(sqlcv1.V1TaskEventTypeFAILED),
			string(sqlcv1.V1TaskEventTypeCANCELLED),
		},
		Eventresourcehints: []string{hint, hint, hint},
	})
	require.NoError(t, err)

	return conditions
}

func TestReplayingARequeuedOperatorStepChildLeavesNoMatchesBehind(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repos := newUserEventScopeTestRepositories(t, pool)
	setOperatorStepDirectCompletion(t, ctx, repos, true)
	tasks := newOperatorStepChildTestTasks(t, ctx, repos, nil)
	orchestrator := createUserEventScopeTestTask(t, ctx, repos, internalTenantId, 303)

	firstEntry, child := spawnOperatorStepChild(t, ctx, repos, orchestrator, 1)
	childExternalId := child.ExternalID
	childTask := TaskIdInsertedAtRetryCount{Id: child.ID, InsertedAt: child.InsertedAt, RetryCount: child.RetryCount}

	_, err := tasks.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{{TaskIdInsertedAtRetryCount: &childTask, Output: []byte(`{"run":1}`)}})
	require.NoError(t, err)

	rerunEntry := reExecuteOperatorStepChildOnNewBranch(t, ctx, repos, orchestrator, firstEntry)
	require.Len(t, listUnsatisfiedConditionsWaitingOnChild(t, ctx, repos, childExternalId), 3, "the new branch entry holds a completion match until the replay decides the child's state")

	_, err = tasks.ReplayTasks(ctx, internalTenantId, []TaskIdInsertedAtRetryCount{childTask})
	require.NoError(t, err)
	require.Empty(t, listUnsatisfiedConditionsWaitingOnChild(t, ctx, repos, childExternalId))

	childTask.RetryCount++
	completed, err := tasks.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{{TaskIdInsertedAtRetryCount: &childTask, Output: []byte(`{"run":2}`)}})
	require.NoError(t, err)
	require.Len(t, completed.SatisfiedDurableEventLogEntries, 1)
	require.Equal(t, rerunEntry.BranchId, completed.SatisfiedDurableEventLogEntries[0].BranchId)
}

// A child replayed straight into FAILED is never finalized, so its FAILED event can only reach
// the orchestrator through the completion match.
func TestReplayingAnOperatorStepChildIntoFailedKeepsItsCompletionMatch(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repos := newUserEventScopeTestRepositories(t, pool)
	setOperatorStepDirectCompletion(t, ctx, repos, true)
	tasks := newOperatorStepChildTestTasks(t, ctx, repos, []CreateConcurrencyOpts{{Expression: "input.key_missing_from_the_input"}})
	orchestrator := createUserEventScopeTestTask(t, ctx, repos, internalTenantId, 304)

	firstEntry, child := spawnOperatorStepChild(t, ctx, repos, orchestrator, 1)
	require.Equal(t, sqlcv1.V1TaskInitialStateFAILED, child.InitialState)

	reExecuteOperatorStepChildOnNewBranch(t, ctx, repos, orchestrator, firstEntry)
	conditionsBeforeReplay := listUnsatisfiedConditionsWaitingOnChild(t, ctx, repos, child.ExternalID)
	require.NotEmpty(t, conditionsBeforeReplay)

	_, err := tasks.ReplayTasks(ctx, internalTenantId, []TaskIdInsertedAtRetryCount{{Id: child.ID, InsertedAt: child.InsertedAt, RetryCount: child.RetryCount}})
	require.NoError(t, err)
	require.Len(t, listUnsatisfiedConditionsWaitingOnChild(t, ctx, repos, child.ExternalID), len(conditionsBeforeReplay))
}

func TestOperatorStepChildWithoutDirectCompletionReportsThroughItsMatch(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repos := newUserEventScopeTestRepositories(t, pool)
	tasks := newOperatorStepChildTestTasks(t, ctx, repos, nil)
	orchestrator := createUserEventScopeTestTask(t, ctx, repos, internalTenantId, 305)

	setOperatorStepDirectCompletion(t, ctx, repos, true)
	_, spawnedWithoutMatch := spawnOperatorStepChild(t, ctx, repos, orchestrator, 1)

	setOperatorStepDirectCompletion(t, ctx, repos, false)
	entryWithMatch, spawnedWithMatch := spawnOperatorStepChild(t, ctx, repos, orchestrator, 2)
	require.Len(t, listUnsatisfiedConditionsWaitingOnChild(t, ctx, repos, spawnedWithMatch.ExternalID), 3)

	completed, err := tasks.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{
		{TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{Id: spawnedWithoutMatch.ID, InsertedAt: spawnedWithoutMatch.InsertedAt}, Output: []byte(`{"child":1}`)},
		{TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{Id: spawnedWithMatch.ID, InsertedAt: spawnedWithMatch.InsertedAt}, Output: []byte(`{"child":2}`)},
	})
	require.NoError(t, err)

	require.Len(t, completed.SatisfiedDurableEventLogEntries, 1, "a child spawned without a match can only report directly")
	require.NotEqual(t, entryWithMatch.NodeId, completed.SatisfiedDurableEventLogEntries[0].NodeId)

	require.Len(t, completed.InternalEvents, 1)
	internalEvent := completed.InternalEvents[0]
	require.Equal(t, spawnedWithMatch.ID, internalEvent.TaskID)

	childHint := spawnedWithMatch.ExternalID.String()
	matchResults, err := repos.matches.ProcessInternalEventMatches(ctx, internalTenantId, []CandidateEventMatch{{
		ID:             uuid.New(),
		EventTimestamp: time.Now(),
		Key:            string(internalEvent.EventType),
		ResourceHint:   &childHint,
		Data:           internalEvent.Data,
	}})
	require.NoError(t, err)
	require.Len(t, matchResults.SatisfiedDurableEventLogEntries, 1)
	require.Equal(t, entryWithMatch.NodeId, matchResults.SatisfiedDurableEventLogEntries[0].NodeId)
	require.Empty(t, listUnsatisfiedConditionsWaitingOnChild(t, ctx, repos, spawnedWithMatch.ExternalID))
}
