//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

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

type operatorStepChildMatchTestEnv struct {
	repo         *TaskRepositoryImpl
	durable      *durableEventsRepository
	orchestrator *sqlcv1.FlattenExternalIdsRow
	workflowName string
	versionId    uuid.UUID
}

// newOperatorStepChildMatchTestEnv triggers a DAG of independent steps through the operator and
// turns direct completion off for the tenant, so spawning a step child writes a completion match.
func newOperatorStepChildMatchTestEnv(t *testing.T, ctx context.Context, pool *pgxpool.Pool, stepIds ...string) operatorStepChildMatchTestEnv {
	t.Helper()

	repo := newBatchTestRepository(pool)
	shared := repo.sharedRepository
	entitlements := newTenantEntitlementRepository(shared)
	durable := newDurableEventsRepository(shared, entitlements, DurableEventBufferOpts{
		FlushInterval:        10 * time.Millisecond,
		MaxBatchSize:         20,
		MaxConcurrentFlushes: 16,
	})

	require.NoError(t, entitlements.SetEntitlements(ctx, internalTenantId, TenantEntitlements{OperatorStepDirectCompletion: false}))

	workflowOpts := minimalWorkflowOpts("operator-step-child-match", "v1", nil)
	workflowOpts.Tasks = nil

	for _, stepId := range stepIds {
		workflowOpts.Tasks = append(workflowOpts.Tasks, CreateStepOpts{ReadableId: stepId, Action: "integration:" + stepId})
	}

	wf, err := (&workflowRepository{sharedRepository: shared}).PutWorkflowVersion(ctx, internalTenantId, workflowOpts)
	require.NoError(t, err)

	tasks, _, _, _, err := newTriggerRepository(shared).TriggerFromWorkflowNames(ctx, internalTenantId, []*WorkflowNameTriggerOpts{{
		TriggerTaskData: &TriggerTaskData{WorkflowName: workflowOpts.Name, Data: []byte(`{}`)},
		ExternalId:      uuid.New(),
	}})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.True(t, tasks[0].IsDagOrchestrator)

	flattened, err := repo.FlattenExternalIds(ctx, internalTenantId, []uuid.UUID{tasks[0].ExternalID})
	require.NoError(t, err)
	require.Len(t, flattened, 1)

	_, err = shared.queries.IncrementLogFileInvocationCounts(ctx, pool, sqlcv1.IncrementLogFileInvocationCountsParams{
		Durabletaskids:         []int64{flattened[0].ID},
		Durabletaskinsertedats: []pgtype.Timestamptz{flattened[0].InsertedAt},
		Tenantids:              []uuid.UUID{internalTenantId},
	})
	require.NoError(t, err)

	return operatorStepChildMatchTestEnv{
		repo:         repo,
		durable:      durable,
		orchestrator: flattened[0],
		workflowName: workflowOpts.Name,
		versionId:    wf.WorkflowVersion.ID,
	}
}

// spawnSteps spawns step children the way the dispatcher does for an orchestrator invocation.
func (e operatorStepChildMatchTestEnv) spawnSteps(t *testing.T, ctx context.Context, invocationCount int32, stepIds ...string) ([]*IngestTriggerRunsEntry, []*V1TaskWithPayload) {
	t.Helper()

	triggerOpts := make([]*WorkflowNameTriggerOpts, len(stepIds))

	for i, stepId := range stepIds {
		actionId := "integration:" + stepId
		childIndex := int64(i)
		parentInsertedAt := e.orchestrator.InsertedAt.Time

		triggerOpts[i] = &WorkflowNameTriggerOpts{
			ReplayOrphanedChildren: true,
			IsDagStepTrigger:       true,
			TriggerTaskData: &TriggerTaskData{
				WorkflowName:         e.workflowName,
				WorkflowVersionId:    &e.versionId,
				TargetActionId:       &actionId,
				Data:                 []byte(`{}`),
				ParentExternalId:     &e.orchestrator.ExternalID,
				ParentTaskId:         &e.orchestrator.ID,
				ParentTaskInsertedAt: &parentInsertedAt,
				ChildIndex:           &childIndex,
				WorkflowRunId:        &e.orchestrator.ExternalID,
			},
		}
	}

	result, err := e.durable.IngestDurableTaskEvent(ctx, IngestDurableTaskEventOpts{
		BaseIngestEventOpts: &BaseIngestEventOpts{
			TenantId:        internalTenantId,
			Task:            e.orchestrator,
			Kind:            sqlcv1.V1DurableEventLogKindRUN,
			InvocationCount: invocationCount,
		},
		TriggerRuns: &IngestTriggerRunsOpts{TriggerOpts: triggerOpts},
	})
	require.NoError(t, err)

	var created []*V1TaskWithPayload

	if pending := result.TriggerRunsResult.PendingTriggers; len(pending) > 0 {
		created, _, _, err = e.durable.TriggerPendingRunEntries(ctx, internalTenantId, []TriggerPendingRunEntriesOpt{{
			Task:        e.orchestrator,
			PendingRuns: pending,
		}})
		require.NoError(t, err)
	}

	return result.TriggerRunsResult.Entries, created
}

// countCompletionMatchRows counts the child's run entry completion matches and their conditions,
// including conditions left without a match. Other matches hinted by the child are not counted.
func countCompletionMatchRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, childExternalId uuid.UUID) (matches, conditions int) {
	t.Helper()

	require.NoError(t, pool.QueryRow(ctx, `
		WITH hinted AS (
			SELECT v1_match_id FROM v1_match_condition WHERE event_resource_hint = $1::text
		), other_matches AS (
			SELECT id FROM v1_match WHERE id IN (SELECT v1_match_id FROM hinted) AND signal_key NOT LIKE 'durable_run:%'
		)
		SELECT
			(SELECT COUNT(*) FROM v1_match WHERE id IN (SELECT v1_match_id FROM hinted) AND signal_key LIKE 'durable_run:%'),
			(SELECT COUNT(*) FROM hinted WHERE v1_match_id NOT IN (SELECT id FROM other_matches))`,
		childExternalId,
	).Scan(&matches, &conditions))

	return matches, conditions
}

func taskRef(task *V1TaskWithPayload) *TaskIdInsertedAtRetryCount {
	return &TaskIdInsertedAtRetryCount{Id: task.ID, InsertedAt: task.InsertedAt, RetryCount: task.RetryCount}
}

func TestOperatorStepChildFinalizeDeletesItsCompletionMatch(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	env := newOperatorStepChildMatchTestEnv(t, ctx, pool, "completes", "fails", "cancels")

	entries, children := env.spawnSteps(t, ctx, 1, "completes", "fails", "cancels")
	require.Len(t, entries, 3)
	require.Len(t, children, 3)

	childByStep := make(map[string]*V1TaskWithPayload, len(children))

	for _, child := range children {
		childByStep[child.StepReadableID] = child

		matches, conditions := countCompletionMatchRows(t, ctx, pool, child.ExternalID)
		require.Equal(t, 1, matches, "with direct completion off, spawning a step child writes its completion match")
		require.Equal(t, 3, conditions)
	}

	completed, err := env.repo.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{{
		TaskIdInsertedAtRetryCount: taskRef(childByStep["completes"]),
		Output:                     []byte(`{"answer":42}`),
	}})
	require.NoError(t, err)
	require.Len(t, completed.SatisfiedDurableEventLogEntries, 1)

	failed, err := env.repo.FailTasks(ctx, internalTenantId, []FailTaskOpts{{
		TaskIdInsertedAtRetryCount: taskRef(childByStep["fails"]),
		IsAppError:                 true,
		ErrorMessage:               "boom",
		IsNonRetryable:             true,
	}})
	require.NoError(t, err)
	require.Len(t, failed.SatisfiedDurableEventLogEntries, 1)

	cancelled, err := env.repo.CancelTasks(ctx, internalTenantId, []TaskIdInsertedAtRetryCount{*taskRef(childByStep["cancels"])})
	require.NoError(t, err)
	require.Len(t, cancelled.SatisfiedDurableEventLogEntries, 1)

	storedEntries, err := env.repo.queries.BulkGetDurableEventLogEntries(ctx, pool, sqlcv1.BulkGetDurableEventLogEntriesParams{
		Durabletaskids:         []int64{env.orchestrator.ID, env.orchestrator.ID, env.orchestrator.ID},
		Durabletaskinsertedats: []pgtype.Timestamptz{env.orchestrator.InsertedAt, env.orchestrator.InsertedAt, env.orchestrator.InsertedAt},
		Branchids:              []int64{entries[0].BranchId, entries[1].BranchId, entries[2].BranchId},
		Nodeids:                []int64{entries[0].NodeId, entries[1].NodeId, entries[2].NodeId},
	})
	require.NoError(t, err)
	require.Len(t, storedEntries, 3)

	for _, entry := range storedEntries {
		require.True(t, entry.IsSatisfied)
	}

	for step, child := range childByStep {
		matches, conditions := countCompletionMatchRows(t, ctx, pool, child.ExternalID)
		require.Zero(t, matches, "step %s left its completion match", step)
		require.Zero(t, conditions, "step %s left its completion match conditions", step)
	}
}

func TestReExecutedOperatorStepChildFinalizeDeletesTheNewBranchMatch(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	env := newOperatorStepChildMatchTestEnv(t, ctx, pool, "reruns", "other")

	firstEntries, children := env.spawnSteps(t, ctx, 1, "reruns")
	require.Len(t, children, 1)
	child := children[0]

	_, err := env.repo.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{{
		TaskIdInsertedAtRetryCount: taskRef(child),
		Output:                     []byte(`{"run":1}`),
	}})
	require.NoError(t, err)

	matches, conditions := countCompletionMatchRows(t, ctx, pool, child.ExternalID)
	require.Zero(t, matches, "the first run left its completion match")
	require.Zero(t, conditions)

	_, err = env.durable.HandleBranchForDAGReplay(ctx, internalTenantId, env.orchestrator, []uuid.UUID{child.ExternalID})
	require.NoError(t, err)

	_, err = env.repo.queries.IncrementLogFileInvocationCounts(ctx, pool, sqlcv1.IncrementLogFileInvocationCountsParams{
		Durabletaskids:         []int64{env.orchestrator.ID},
		Durabletaskinsertedats: []pgtype.Timestamptz{env.orchestrator.InsertedAt},
		Tenantids:              []uuid.UUID{internalTenantId},
	})
	require.NoError(t, err)

	rerunEntries, rerunChildren := env.spawnSteps(t, ctx, 2, "reruns")
	require.Len(t, rerunEntries, 1)
	require.Empty(t, rerunChildren, "a re-executed child keeps its task")
	require.True(t, rerunEntries[0].ChildNeedsReplay)
	require.Equal(t, child.ExternalID, rerunEntries[0].WorkflowRunExternalId)
	require.NotEqual(t, firstEntries[0].BranchId, rerunEntries[0].BranchId)

	matches, conditions = countCompletionMatchRows(t, ctx, pool, child.ExternalID)
	require.Equal(t, 1, matches, "the new branch entry gets its own completion match")
	require.Equal(t, 3, conditions)

	replayed, err := env.repo.ReplayTasks(ctx, internalTenantId, []TaskIdInsertedAtRetryCount{*taskRef(child)})
	require.NoError(t, err)
	require.Len(t, replayed.ReplayedTasks, 1)

	completed, err := env.repo.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{{
		TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{Id: child.ID, InsertedAt: child.InsertedAt, RetryCount: child.RetryCount + 1},
		Output:                     []byte(`{"run":2}`),
	}})
	require.NoError(t, err)
	require.Len(t, completed.SatisfiedDurableEventLogEntries, 1)
	require.Equal(t, rerunEntries[0].BranchId, completed.SatisfiedDurableEventLogEntries[0].BranchId)

	matches, conditions = countCompletionMatchRows(t, ctx, pool, child.ExternalID)
	require.Zero(t, matches)
	require.Zero(t, conditions)
}
