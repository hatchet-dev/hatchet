//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/enums"
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

	singleTaskChildRun := &sqlcv1.ReleaseTasksRow{
		ExternalID:           otherExternalId,
		WorkflowRunID:        otherExternalId,
		ParentTaskExternalID: &orchestratorExternalId,
		ParentTaskID:         pgtype.Int8{Int64: 1, Valid: true},
		ParentTaskInsertedAt: pgtype.Timestamptz{Valid: true},
	}
	require.True(t, IsOperatorStepChild(singleTaskChildRun))

	rootTask := &sqlcv1.ReleaseTasksRow{ExternalID: otherExternalId, WorkflowRunID: otherExternalId}
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

func TestDurableParentsSingleTaskChildHasNoCompletionMatchWithDirectSpawn(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repos := newUserEventScopeTestRepositories(t, pool)
	repos.shared.childRunDirectSpawn = true

	workflows := &workflowRepository{sharedRepository: repos.shared}
	_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("durable-child", "v1", nil))
	require.NoError(t, err)

	parent := createUserEventScopeTestTask(t, ctx, repos, internalTenantId, 501)
	parentInsertedAt := parent.InsertedAt.Time
	childIndex := int64(0)

	ingested, err := repos.durable.IngestDurableTaskEvent(ctx, IngestDurableTaskEventOpts{
		BaseIngestEventOpts: &BaseIngestEventOpts{
			Task:            parent,
			Kind:            enums.V1DurableEventLogKindRUN,
			InvocationCount: 1,
			TenantId:        internalTenantId,
		},
		TriggerRuns: &IngestTriggerRunsOpts{
			TriggerOpts: []*WorkflowNameTriggerOpts{{
				TriggerTaskData: &TriggerTaskData{
					WorkflowName:         "durable-child",
					Data:                 []byte(`{}`),
					ParentExternalId:     &parent.ExternalID,
					ParentTaskId:         &parent.ID,
					ParentTaskInsertedAt: &parentInsertedAt,
					ChildIndex:           &childIndex,
				},
			}},
		},
	})
	require.NoError(t, err)

	createdTasks, _, _, err := repos.durable.TriggerPendingRunEntries(ctx, internalTenantId, []TriggerPendingRunEntriesOpt{{
		Task:        parent,
		PendingRuns: ingested.TriggerRunsResult.PendingTriggers,
	}})
	require.NoError(t, err)
	require.Len(t, createdTasks, 1)

	childHint := createdTasks[0].ExternalID.String()
	completionConditions, err := repos.shared.queries.ListMatchConditionsForEventWithHint(ctx, pool, sqlcv1.ListMatchConditionsForEventWithHintParams{
		Tenantid:           internalTenantId,
		Eventtype:          sqlcv1.V1EventTypeINTERNAL,
		Eventkeys:          []string{string(sqlcv1.V1TaskEventTypeCOMPLETED)},
		Eventresourcehints: []string{childHint},
	})
	require.NoError(t, err)
	require.Empty(t, completionConditions)
}

func TestChildSpawnedBeforeDeterministicIdsKeepsItsRecordedId(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	repos := newUserEventScopeTestRepositories(t, pool)

	parentSpawningBeforeEnablement := createUserEventScopeTestTask(t, ctx, repos, tenantID, 401)
	childWithRecordedId := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, parentSpawningBeforeEnablement, 1, "my-dag:step-a", 1)

	repos.shared.childRunDirectSpawn = true

	respawnedChild := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, parentSpawningBeforeEnablement, 1, "my-dag:step-a", 1)
	require.True(t, respawnedChild.AlreadyExisted)
	require.Equal(t, childWithRecordedId.WorkflowRunExternalId, respawnedChild.WorkflowRunExternalId)

	parentSpawningAfterEnablement := createUserEventScopeTestTask(t, ctx, repos, tenantID, 402)
	childWithDeterministicId := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, parentSpawningAfterEnablement, 1, "my-dag:step-a", 1)

	spawnKey := getChildSignalEventKey(parentSpawningAfterEnablement.ExternalID, dagStepSpawnStepIndex, 1, nil)
	require.Equal(t, deterministicChildExternalId(parentSpawningAfterEnablement.ExternalID, spawnKey), childWithDeterministicId.WorkflowRunExternalId)

	respawnedDeterministicChild := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, parentSpawningAfterEnablement, 1, "my-dag:step-a", 1)
	require.True(t, respawnedDeterministicChild.AlreadyExisted)
	require.Equal(t, childWithDeterministicId.WorkflowRunExternalId, respawnedDeterministicChild.WorkflowRunExternalId)
}

func TestChildSpawnedWithDeterministicIdKeepsItAfterDisablement(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	tenantID := uuid.New()
	repos := newUserEventScopeTestRepositories(t, pool)

	repos.shared.childRunDirectSpawn = true

	parent := createUserEventScopeTestTask(t, ctx, repos, tenantID, 403)
	childWithDeterministicId := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, parent, 1, "my-dag:step-a", 1)

	repos.shared.childRunDirectSpawn = false

	respawnedChild := ingestDagStepTrigger(t, ctx, repos.durable, tenantID, parent, 1, "my-dag:step-a", 1)
	require.True(t, respawnedChild.AlreadyExisted)
	require.Equal(t, childWithDeterministicId.WorkflowRunExternalId, respawnedChild.WorkflowRunExternalId)
}
