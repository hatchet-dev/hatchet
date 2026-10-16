//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggeredDagStoresItsOwnInputPayloadWhenWritingPerRun(t *testing.T) {
	tasks, dag, shared := triggerTwoStepDag(t, true)

	assert.Nil(t, dag.PayloadReferenceID)

	for _, task := range tasks {
		assert.Nil(t, task.PayloadReferenceID)
		assertTaskInputIsStored(t, shared, task)
	}
}

func TestTriggeredDagSharesItsInputPayloadWithItsTasksWhenNotWritingPerRun(t *testing.T) {
	tasks, dag, shared := triggerTwoStepDag(t, false)

	require.NotNil(t, dag.PayloadReferenceID)

	for _, task := range tasks {
		require.NotNil(t, task.PayloadReferenceID)
		assert.Equal(t, *dag.PayloadReferenceID, *task.PayloadReferenceID)
		assertTaskInputIsStored(t, shared, task)
		assertTaskHasNoOwnInputRow(t, shared, task)
	}
}

func triggerTwoStepDag(t *testing.T, writeInputPayloadPerRun bool) ([]*V1TaskWithPayload, *DAGWithData, *sharedRepository) {
	t.Helper()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(func() { cleanup() })

	shared := newBatchTestRepository(pool).sharedRepository
	shared.writeInputPayloadPerRun = writeInputPayloadPerRun

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

	assert.JSONEq(t, `{"order_id": 1}`, string(UserInputFromTaskInputPayload(stored.Bytes())))
}
