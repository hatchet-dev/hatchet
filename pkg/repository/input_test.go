//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestNewV1StepRunDataFromTaskAndInputPayload_BuildsFromTheBareInputAndTheTaskColumns(t *testing.T) {
	dagParentTaskRunId := uuid.New()

	task := &sqlcv1.V1Task{
		Triggers:                     []byte(`{"filter_payload": {"region": "eu"}, "wait_for_event:go": {"order_id": 1}}`),
		DagParentTaskRunIds:          []uuid.UUID{dagParentTaskRunId},
		ChildTaskDesiredWorkerLabels: []byte(`[{"key": "pool"}]`),
		TaskInputPayloadVersion:      int16(TaskInputPayloadVersionUserInputOnly),
	}

	data, err := NewV1StepRunDataFromTaskAndInputPayload(task, []byte(`{"order_id": 1}`))
	require.NoError(t, err)

	assert.JSONEq(t, `{"order_id": 1}`, string(data.Input))
	assert.Empty(t, data.Parents)
	assert.Equal(t, map[string]interface{}{"region": "eu"}, data.Triggers["filter_payload"])
	assert.Equal(t, map[string]interface{}{"order_id": float64(1)}, data.Triggers["wait_for_event:go"])
	assert.Equal(t, []uuid.UUID{dagParentTaskRunId}, data.DagParentTaskRunIds)
	require.Len(t, data.DesiredWorkerLabels, 1)
	assert.Equal(t, "pool", data.DesiredWorkerLabels[0].Key)
}

func TestNewV1StepRunDataFromTaskAndInputPayload_SendsTheSameKeysAsBeforeWhenTheColumnsAreEmpty(t *testing.T) {
	task := &sqlcv1.V1Task{TaskInputPayloadVersion: int16(TaskInputPayloadVersionUserInputOnly)}

	data, err := NewV1StepRunDataFromTaskAndInputPayload(task, []byte(`{"order_id": 1}`))
	require.NoError(t, err)

	assert.JSONEq(t, `{"input": {"order_id": 1}, "parents": {}, "triggers": {"filter_payload": null}}`, string(data.Bytes()))
}

func TestNewV1StepRunDataFromTaskAndInputPayload_ReadsARowStoredBeforeTheColumnsExisted(t *testing.T) {
	legacyTaskInputPayload := []byte(`{
		"input": {"order_id": 1},
		"parents": {"step1": {"result": 2}},
		"triggers": {"filter_payload": {"region": "eu"}},
		"triggered_by": "manual",
		"user_data": null,
		"overrides": null
	}`)

	task := &sqlcv1.V1Task{TaskInputPayloadVersion: int16(TaskInputPayloadVersionStepRunData)}

	data, err := NewV1StepRunDataFromTaskAndInputPayload(task, legacyTaskInputPayload)
	require.NoError(t, err)

	assert.JSONEq(t, `{"order_id": 1}`, string(data.Input))
	assert.Equal(t, map[string]interface{}{"result": float64(2)}, data.Parents["step1"])
	assert.Equal(t, map[string]interface{}{"region": "eu"}, data.Triggers["filter_payload"])
}

func TestNewTaskInputFromTaskInputPayload_CarriesTheLegacyColumnsSoAReplayDoesNotLoseThem(t *testing.T) {
	dagParentTaskRunId := uuid.New()
	legacyTaskInputPayload := []byte(`{
		"input": {"order_id": 1},
		"parents": {},
		"triggers": {"filter_payload": null},
		"dag_parent_task_run_ids": ["` + dagParentTaskRunId.String() + `"],
		"desired_worker_labels": [{"key": "pool"}]
	}`)

	logger := zerolog.Nop()
	taskInput := (&sharedRepository{l: &logger}).newTaskInputFromTaskInputPayload(TaskInputPayloadVersionStepRunData, legacyTaskInputPayload)

	assert.Equal(t, map[string]interface{}{"order_id": float64(1)}, taskInput.Input)
	assert.Equal(t, []uuid.UUID{dagParentTaskRunId}, taskInput.DagParentTaskRunIds)
	require.Len(t, taskInput.DesiredWorkerLabels, 1)
	assert.Equal(t, "pool", taskInput.DesiredWorkerLabels[0].Key)
}

func TestCreatedTaskStoresOnlyTheUserInputAndKeepsTheRestOnTheTask(t *testing.T) {
	_, inserted, task := insertTaskWithTriggersAndDagParents(t, false)

	assert.JSONEq(t, `{"order_id": 1}`, string(inserted.TaskInputPayload))
	assert.Equal(t, int16(TaskInputPayloadVersionUserInputOnly), task.TaskInputPayloadVersion)

	assertStepRunDataReadsBackTheTaskInput(t, task, inserted.TaskInputPayload)
}

func TestCreatedTaskStoresTheWrappedStepRunDataWhenWrappingIsOn(t *testing.T) {
	_, inserted, task := insertTaskWithTriggersAndDagParents(t, true)

	assert.JSONEq(t, string(inserted.Payload), string(inserted.TaskInputPayload))
	assert.Equal(t, int16(TaskInputPayloadVersionStepRunData), task.TaskInputPayloadVersion)

	assertStepRunDataReadsBackTheTaskInput(t, task, inserted.TaskInputPayload)
}

func TestReplayKeepsTheTaskInputPayloadVersionTheTaskWasCreatedWith(t *testing.T) {
	for _, wrapTaskInputInStepRunDataAtCreation := range []bool{true, false} {
		shared, inserted, task := insertTaskWithTriggersAndDagParents(t, wrapTaskInputInStepRunDataAtCreation)
		shared.wrapTaskInputInStepRunData = !wrapTaskInputInStepRunDataAtCreation

		version := TaskInputPayloadVersion(task.TaskInputPayloadVersion)
		replayed := replayTaskWithInput(t, shared, task, shared.newTaskInputFromTaskInputPayload(version, inserted.TaskInputPayload))

		replayedTasks, err := shared.listTasks(context.Background(), shared.pool, internalTenantId, []int64{task.ID})
		require.NoError(t, err)
		require.Len(t, replayedTasks, 1)

		assert.Equal(t, task.TaskInputPayloadVersion, replayedTasks[0].TaskInputPayloadVersion)

		data, err := NewV1StepRunDataFromTaskAndInputPayload(replayedTasks[0], replayed.TaskInputPayload)
		require.NoError(t, err)

		assert.JSONEq(t, `{"order_id": 1}`, string(data.Input))
		assert.Equal(t, task.DagParentTaskRunIds, data.DagParentTaskRunIds)
		require.Len(t, data.DesiredWorkerLabels, 1)
		assert.Equal(t, "pool", data.DesiredWorkerLabels[0].Key)
	}
}

func replayTaskWithInput(t *testing.T, shared *sharedRepository, task *sqlcv1.V1Task, input *TaskInput) *V1TaskWithPayload {
	t.Helper()

	replayed, err := shared.replayTasks(context.Background(), shared.pool, internalTenantId, []ReplayTaskOpts{
		{
			TaskId:       task.ID,
			InsertedAt:   task.InsertedAt,
			ExternalId:   task.ExternalID,
			StepId:       task.StepID,
			Input:        input,
			InitialState: sqlcv1.V1TaskInitialStateQUEUED,
		},
	})
	require.NoError(t, err)
	require.Len(t, replayed, 1)

	return replayed[0]
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

	assert.JSONEq(t, `{"filter_payload": {"region": "eu"}}`, string(tasks[0].Triggers))
	assert.Equal(t, dagParentTaskRunIds, tasks[0].DagParentTaskRunIds)

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
