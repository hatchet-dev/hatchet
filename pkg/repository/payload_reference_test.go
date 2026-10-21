//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggeredDagStoresItsOwnInputPayloadWithoutPayloadConsolidation(t *testing.T) {
	tasks, dag, shared := triggerTwoStepDag(t, false)

	assert.Nil(t, dag.PayloadReferenceID)

	for _, task := range tasks {
		assert.Nil(t, task.PayloadReferenceID)
		assertTaskInputIsStored(t, shared, task)
	}
}

func TestTriggeredDagSharesItsInputPayloadWithItsTasksWithPayloadConsolidation(t *testing.T) {
	tasks, dag, shared := triggerTwoStepDag(t, true)

	require.NotNil(t, dag.PayloadReferenceID)

	for _, task := range tasks {
		require.NotNil(t, task.PayloadReferenceID)
		assert.Equal(t, *dag.PayloadReferenceID, *task.PayloadReferenceID)
		assertTaskInputIsStored(t, shared, task)
		assertTaskHasNoOwnInputRow(t, shared, task)
	}
}

func TestEventTriggeredOperatorDagOrchestratorSharesTheEventInputPayload(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(func() { cleanup() })

	shared := newBatchTestRepository(pool).sharedRepository
	shared.enablePayloadConsolidation = true

	ctx := context.Background()

	require.NoError(t, newTenantEntitlementRepository(shared).SetEntitlements(ctx, internalTenantId, TenantEntitlements{DAGOperator: true}))

	const eventKey = "order:created"

	workflows := &workflowRepository{sharedRepository: shared}
	workflowOpts := minimalWorkflowOpts("payload-reference-operator-dag", "v1", nil)
	workflowOpts.EventTriggers = []string{eventKey}
	workflowOpts.Tasks = append(workflowOpts.Tasks, CreateStepOpts{
		ReadableId: "step2",
		Action:     "integration:step2",
		Parents:    []string{"step1"},
	})

	_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, workflowOpts)
	require.NoError(t, err)

	eventPayloadReferenceId := NewPayloadReferenceId()

	result, err := newTriggerRepository(shared).TriggerFromEvents(ctx, internalTenantId, []EventTriggerOpts{
		{
			ExternalId:         uuid.New(),
			SeenAt:             time.Now(),
			Key:                eventKey,
			Data:               []byte(`{"order_id": 1}`),
			PayloadReferenceId: &eventPayloadReferenceId,
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Tasks, 1, "only the orchestrator is created at trigger time")

	orchestrator := result.Tasks[0]
	require.True(t, orchestrator.IsDagOrchestrator)
	require.NotNil(t, orchestrator.PayloadReferenceID)
	assert.Equal(t, eventPayloadReferenceId, *orchestrator.PayloadReferenceID)

	assertTaskInputIsStored(t, shared, orchestrator)
	assertTaskHasNoOwnInputRow(t, shared, orchestrator)
}

func triggerTwoStepDag(t *testing.T, enablePayloadConsolidation bool) ([]*V1TaskWithPayload, *DAGWithData, *sharedRepository) {
	t.Helper()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(func() { cleanup() })

	shared := newBatchTestRepository(pool).sharedRepository
	shared.enablePayloadConsolidation = enablePayloadConsolidation

	ctx := context.Background()

	workflows := &workflowRepository{sharedRepository: shared}
	workflowOpts := minimalWorkflowOpts("payload-reference-dag", "v1", nil)
	workflowOpts.Tasks = append(workflowOpts.Tasks, CreateStepOpts{
		ReadableId: "step2",
		Action:     "integration:step2",
		Parents:    []string{"step1"},
	})

	_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, workflowOpts)
	require.NoError(t, err)

	triggers := newTriggerRepository(shared)

	tasks, dags, _, _, err := triggers.TriggerFromWorkflowNames(ctx, internalTenantId, []*WorkflowNameTriggerOpts{
		{
			TriggerTaskData: &TriggerTaskData{
				WorkflowName: workflowOpts.Name,
				Data:         []byte(`{"order_id": 1}`),
			},
			ExternalId: uuid.New(),
		},
	})
	require.NoError(t, err)
	require.Len(t, dags, 1)
	require.Len(t, tasks, 1, "step2 waits on step1, so only the root task is created at trigger time")

	return tasks, dags[0], shared
}

func assertTaskHasNoOwnInputRow(t *testing.T, shared *sharedRepository, task *V1TaskWithPayload) {
	t.Helper()

	ownRow, err := shared.payloadStore.TaskInputs().RetrieveOne(context.Background(), shared.pool, PayloadRef[TaskInputShape]{
		Id:         task.ID,
		InsertedAt: task.InsertedAt,
		TenantId:   internalTenantId,
		ExternalId: task.ExternalID,
	})
	require.NoError(t, err)

	assert.True(t, ownRow.IsEmpty(), "a task that references its run's row must not store its own")
}

func assertTaskInputIsStored(t *testing.T, shared *sharedRepository, task *V1TaskWithPayload) {
	t.Helper()

	stored, err := shared.payloadStore.TaskInputs().RetrieveOne(context.Background(), shared.pool, PayloadRef[TaskInputShape]{
		Id:                 task.ID,
		InsertedAt:         task.InsertedAt,
		TenantId:           internalTenantId,
		ExternalId:         task.ExternalID,
		PayloadReferenceId: task.PayloadReferenceID,
	})
	require.NoError(t, err)

	userInput, err := userInputFromTaskInputPayload(taskInputPayloadVersion(task.TaskInputPayloadVersion), stored.Bytes())
	require.NoError(t, err)

	assert.JSONEq(t, `{"order_id": 1}`, string(userInput))
}

func TestCompletedTaskStoresItsOutputUnderThePayloadReferenceId(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(func() { cleanup() })

	repo := newBatchTestRepository(pool)
	shared := repo.sharedRepository
	ctx := context.Background()

	workflows := &workflowRepository{sharedRepository: shared}
	workflowOpts := minimalWorkflowOpts("payload-reference-output", "v1", nil)

	_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, workflowOpts)
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
	outputPayloadReferenceId := NewPayloadReferenceId()

	_, err = repo.CompleteTasks(ctx, internalTenantId, []CompleteTaskOpts{
		{
			TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				RetryCount: task.RetryCount,
			},
			Output:             []byte(`{"result": 42}`),
			PayloadReferenceId: &outputPayloadReferenceId,
		},
	})
	require.NoError(t, err)

	outputEvents, err := shared.queries.ListTaskOutputEventIdsByTaskRunExternalIds(ctx, pool, []uuid.UUID{task.ExternalID})
	require.NoError(t, err)
	require.Len(t, outputEvents, 1)
	require.NotNil(t, outputEvents[0].OutputEventPayloadReferenceID)
	assert.Equal(t, outputPayloadReferenceId, *outputEvents[0].OutputEventPayloadReferenceID)

	byReference, err := shared.payloadStore.TaskCompletionEvents().RetrieveOne(ctx, pool, PayloadRef[TaskCompletionEventShape]{
		TenantId:           internalTenantId,
		PayloadReferenceId: &outputPayloadReferenceId,
	})
	require.NoError(t, err)

	completionEvent, err := DecodeTaskCompletionEventPayload(byReference)
	require.NoError(t, err)
	assert.JSONEq(t, `{"result": 42}`, string(completionEvent.Output), "the completion event row is stored under the reference id")

	coreOutputs, err := repo.listTaskOutputEvents(ctx, pool, internalTenantId, []uuid.UUID{task.ExternalID})
	require.NoError(t, err)
	require.Len(t, coreOutputs, 1)
	assert.JSONEq(t, `{"result": 42}`, string(coreOutputs[0].Output))

	olapEventExternalId := uuid.New()

	olapPayloads, err := createOLAPRepositoryWithPayloadStore(t, pool).readPayloads(ctx, internalTenantId, ReadOLAPPayloadOpts{
		ExternalId:         olapEventExternalId,
		InsertedAt:         outputEvents[0].TaskEventInsertedAt,
		PayloadReferenceId: &outputPayloadReferenceId,
		Kind:               PayloadKindTaskOutput,
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"result": 42}`, string(olapPayloads[olapEventExternalId]), "OLAP reads the bare output out of the referenced completion event")
}
