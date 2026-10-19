//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestNewV1StepRunDataFromTaskAndInputPayload_SendsTheSameKeysAsBeforeWhenTheColumnsAreEmpty(t *testing.T) {
	task := &sqlcv1.V1Task{TaskInputPayloadVersion: int16(taskInputPayloadVersionUserInputOnly)}

	data, err := NewV1StepRunDataFromTaskAndInputPayload(task, []byte(`{"order_id": 1}`))
	require.NoError(t, err)

	assert.JSONEq(t, `{"input": {"order_id": 1}, "parents": {}, "triggers": {"filter_payload": null}}`, string(data.Bytes()))
}

func TestTaskInputReadsBackAfterCreateAndAfterReplayingWithTheFlagFlipped(t *testing.T) {
	for wrapTaskInputInStepRunDataAtCreation, expectedVersion := range map[bool]taskInputPayloadVersion{
		true:  taskInputPayloadVersionStepRunData,
		false: taskInputPayloadVersionUserInputOnly,
	} {
		shared, inserted, task := insertTaskWithTriggersAndDagParents(t, wrapTaskInputInStepRunDataAtCreation)

		assert.Equal(t, int16(expectedVersion), task.TaskInputPayloadVersion)
		assertStepRunDataReadsBackTheTaskInput(t, task, inserted.TaskInputPayload)

		shared.wrapTaskInputInStepRunData = !wrapTaskInputInStepRunDataAtCreation

		replayed, err := shared.replayTasks(context.Background(), shared.pool, internalTenantId, []ReplayTaskOpts{
			{
				TaskId:       task.ID,
				InsertedAt:   task.InsertedAt,
				ExternalId:   task.ExternalID,
				StepId:       task.StepID,
				Input:        shared.newTaskInputFromTaskInputPayload(expectedVersion, inserted.TaskInputPayload),
				InitialState: sqlcv1.V1TaskInitialStateQUEUED,
			},
		})
		require.NoError(t, err)
		require.Len(t, replayed, 1)

		replayedTasks, err := shared.listTasks(context.Background(), shared.pool, internalTenantId, []int64{task.ID})
		require.NoError(t, err)
		require.Len(t, replayedTasks, 1)

		assert.Equal(t, int16(expectedVersion), replayedTasks[0].TaskInputPayloadVersion)

		data, err := NewV1StepRunDataFromTaskAndInputPayload(replayedTasks[0], replayed[0].TaskInputPayload)
		require.NoError(t, err)

		assert.JSONEq(t, `{"order_id": 1}`, string(data.Input))
		assert.Equal(t, task.DagParentTaskRunIds, data.DagParentTaskRunIds)
		require.Len(t, data.DesiredWorkerLabels, 1)
		assert.Equal(t, "pool", data.DesiredWorkerLabels[0].Key)
	}
}

func insertTaskWithTriggersAndDagParents(t *testing.T, wrapTaskInputInStepRunData bool) (*sharedRepository, *V1TaskWithPayload, *sqlcv1.V1Task) {
	t.Helper()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(func() { cleanup() })

	repo := newBatchTestRepository(pool)
	shared := repo.sharedRepository
	shared.wrapTaskInputInStepRunData = wrapTaskInputInStepRunData
	workflows := &workflowRepository{sharedRepository: shared}

	ctx := context.Background()

	wf, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("stored-task-input", "v1", nil))
	require.NoError(t, err)

	steps, err := repo.queries.ListStepsByWorkflowVersionIds(ctx, pool, sqlcv1.ListStepsByWorkflowVersionIdsParams{
		Ids:      []uuid.UUID{wf.WorkflowVersion.ID},
		Tenantid: internalTenantId,
	})
	require.NoError(t, err)
	require.Len(t, steps, 1)

	stepIdsToConfig, err := shared.listStepsByIds(ctx, pool, internalTenantId, []uuid.UUID{steps[0].ID})
	require.NoError(t, err)

	dagParentTaskRunIds := []uuid.UUID{uuid.New(), uuid.New()}

	taskInput := shared.newTaskInput([]byte(`{"order_id": 1}`), nil, []byte(`{"region": "eu"}`), dagParentTaskRunIds)
	taskInput.DesiredWorkerLabels = []*sqlcv1.GetDesiredLabelsRow{{Key: "pool"}}

	inserted, err := shared.insertTasks(ctx, pool, internalTenantId, []CreateTaskOpts{
		{
			ExternalId:    uuid.New(),
			WorkflowRunId: uuid.New(),
			StepId:        steps[0].ID,
			StepIndex:     0,
			InitialState:  sqlcv1.V1TaskInitialStateQUEUED,
			Input:         taskInput,
		},
	}, stepIdsToConfig)
	require.NoError(t, err)
	require.Len(t, inserted, 1)

	tasks, err := shared.listTasks(ctx, pool, internalTenantId, []int64{inserted[0].ID})
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	return shared, inserted[0], tasks[0]
}

func assertStepRunDataReadsBackTheTaskInput(t *testing.T, task *sqlcv1.V1Task, taskInputPayload []byte) {
	t.Helper()

	data, err := NewV1StepRunDataFromTaskAndInputPayload(task, taskInputPayload)
	require.NoError(t, err)

	assert.JSONEq(t, `{"order_id": 1}`, string(data.Input))
	assert.Equal(t, map[string]interface{}{"region": "eu"}, data.Triggers["filter_payload"])
	assert.Len(t, data.DagParentTaskRunIds, 2)
	assert.Equal(t, task.DagParentTaskRunIds, data.DagParentTaskRunIds)
	require.Len(t, data.DesiredWorkerLabels, 1)
	assert.Equal(t, "pool", data.DesiredWorkerLabels[0].Key)
}
