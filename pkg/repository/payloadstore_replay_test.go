//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// A task created directly in a terminal state (e.g. cancelled because its parent failed) has no
// TASK_INPUT row. The overwrite on replay must insert when the row is missing so the recomputed
// input is what gets read.
func TestReplayOverwritesOffloadedTaskInput(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := newBatchTestRepository(pool)
	shared := repo.sharedRepository
	workflows := &workflowRepository{sharedRepository: shared}

	ctx := context.Background()

	wf, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("replay-offloaded-input", "v1", nil))
	require.NoError(t, err)

	steps, err := repo.queries.ListStepsByWorkflowVersionIds(ctx, pool, sqlcv1.ListStepsByWorkflowVersionIdsParams{
		Ids:      []uuid.UUID{wf.WorkflowVersion.ID},
		Tenantid: internalTenantId,
	})
	require.NoError(t, err)
	require.Len(t, steps, 1)

	stepID := steps[0].ID

	stepIdsToConfig, err := shared.listStepsByIds(ctx, pool, internalTenantId, []uuid.UUID{stepID})
	require.NoError(t, err)

	// Mirror the match CANCEL branch: the task is created without an Input.
	inserted, err := shared.insertTasks(ctx, pool, internalTenantId, []CreateTaskOpts{
		{
			ExternalId:    uuid.New(),
			WorkflowRunId: uuid.New(),
			StepId:        stepID,
			StepIndex:     0,
			InitialState:  sqlcv1.V1TaskInitialStateCANCELLED,
		},
	}, stepIdsToConfig)
	require.NoError(t, err)
	require.Len(t, inserted, 1)

	task := inserted[0]

	// Mirror processEventMatches, which stores whatever payload the created task carries.
	err = shared.payloadStore.Store(ctx, pool, StorePayloadOpts{
		Id:         task.ID,
		InsertedAt: task.InsertedAt,
		ExternalId: task.ExternalID,
		Type:       sqlcv1.V1PayloadTypeTASKINPUT,
		Payload:    task.TaskInputPayload,
		TenantId:   internalTenantId,
	})
	require.NoError(t, err)

	retrieveOpt := RetrievePayloadOpts{
		Id:         task.ID,
		InsertedAt: task.InsertedAt,
		Type:       sqlcv1.V1PayloadTypeTASKINPUT,
		TenantId:   internalTenantId,
		ExternalId: task.ExternalID,
	}

	missing, err := shared.payloadStore.RetrieveSingle(ctx, pool, retrieveOpt)
	require.NoError(t, err)
	require.Nil(t, missing, "a task created in a terminal state without input has no TASK_INPUT row")

	replayInput := &TaskInput{Input: map[string]interface{}{"case_id": "replayed"}}

	replayed, err := shared.replayTasks(ctx, pool, internalTenantId, []ReplayTaskOpts{
		{
			TaskId:       task.ID,
			InsertedAt:   task.InsertedAt,
			ExternalId:   task.ExternalID,
			StepId:       stepID,
			Input:        replayInput,
			InitialState: sqlcv1.V1TaskInitialStateQUEUED,
		},
	})
	require.NoError(t, err)
	require.Len(t, replayed, 1)

	afterReplay, err := shared.payloadStore.RetrieveSingle(ctx, pool, retrieveOpt)
	require.NoError(t, err)
	require.NotNil(t, afterReplay, "replay must persist the recomputed input even when the original row was offloaded")

	var storedInput map[string]interface{}
	require.NoError(t, json.Unmarshal(afterReplay, &storedInput))
	require.Equal(t, "replayed", storedInput["case_id"])

	// A second replay while the row exists must still update in place.
	replayed, err = shared.replayTasks(ctx, pool, internalTenantId, []ReplayTaskOpts{
		{
			TaskId:       task.ID,
			InsertedAt:   task.InsertedAt,
			ExternalId:   task.ExternalID,
			StepId:       stepID,
			Input:        &TaskInput{Input: map[string]interface{}{"case_id": "replayed-again"}},
			InitialState: sqlcv1.V1TaskInitialStateQUEUED,
		},
	})
	require.NoError(t, err)
	require.Len(t, replayed, 1)

	afterSecondReplay, err := shared.payloadStore.RetrieveSingle(ctx, pool, retrieveOpt)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(afterSecondReplay, &storedInput))
	require.Equal(t, "replayed-again", storedInput["case_id"])

	var rowCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM v1_payload WHERE tenant_id = $1 AND id = $2 AND type = 'TASK_INPUT'`,
		internalTenantId, task.ID).Scan(&rowCount))
	require.Equal(t, 1, rowCount)
}
