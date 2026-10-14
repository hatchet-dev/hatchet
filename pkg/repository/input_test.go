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
	data, err := NewV1StepRunDataFromTaskAndInputPayload(&sqlcv1.V1Task{}, []byte(`{"order_id": 1}`))
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

	data, err := NewV1StepRunDataFromTaskAndInputPayload(&sqlcv1.V1Task{}, legacyTaskInputPayload)
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
	taskInput := (&sharedRepository{l: &logger}).newTaskInputFromTaskInputPayload(legacyTaskInputPayload)

	assert.Equal(t, map[string]interface{}{"order_id": float64(1)}, taskInput.Input)
	assert.Equal(t, []uuid.UUID{dagParentTaskRunId}, taskInput.DagParentTaskRunIds)
	require.Len(t, taskInput.DesiredWorkerLabels, 1)
	assert.Equal(t, "pool", taskInput.DesiredWorkerLabels[0].Key)
}

func TestUserInputFromTaskInputPayload_DoesNotUnwrapAUserInputThatHasItsOwnInputKey(t *testing.T) {
	userInput := []byte(`{"input": "hello", "parents": ["a", "b"]}`)

	assert.JSONEq(t, string(userInput), string(UserInputFromTaskInputPayload(userInput)))
}

func TestCreatedTaskStoresOnlyTheUserInputAndKeepsTheRestOnTheTask(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := newBatchTestRepository(pool)
	shared := repo.sharedRepository
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

	assert.JSONEq(t, `{"order_id": 1}`, string(inserted[0].UserInput))

	tasks, err := shared.listTasks(ctx, pool, internalTenantId, []int64{inserted[0].ID})
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	assert.JSONEq(t, `{"filter_payload": {"region": "eu"}}`, string(tasks[0].Triggers))
	assert.Equal(t, dagParentTaskRunIds, tasks[0].DagParentTaskRunIds)

	data, err := NewV1StepRunDataFromTaskAndInputPayload(tasks[0], inserted[0].UserInput)
	require.NoError(t, err)

	assert.JSONEq(t, `{"order_id": 1}`, string(data.Input))
	assert.Equal(t, map[string]interface{}{"region": "eu"}, data.Triggers["filter_payload"])
	assert.Equal(t, dagParentTaskRunIds, data.DagParentTaskRunIds)
	require.Len(t, data.DesiredWorkerLabels, 1)
	assert.Equal(t, "pool", data.DesiredWorkerLabels[0].Key)
}
