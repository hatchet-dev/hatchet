package workflowruns

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func TestConcurrencySlotExternalIdsValidation(t *testing.T) {
	sharedName := "per-customer"
	empty := ""
	workflowId := uuid.New()
	taskName := "process"
	key := "alpha"
	badState := gen.V1ConcurrencySlotState("BOGUS")
	holding := gen.V1ConcurrencySlotStateHolding

	tests := []struct {
		name    string
		filters workflowRunFilters
		wantErr bool
	}{
		{name: "unset", filters: workflowRunFilters{}},
		{name: "key only", filters: workflowRunFilters{ConcurrencyKey: &key}, wantErr: true},
		{name: "strategy without key", filters: workflowRunFilters{ConcurrencyWorkflowId: &workflowId, ConcurrencyTaskName: &taskName}, wantErr: true},
		{name: "state only", filters: workflowRunFilters{ConcurrencySlotState: &holding}, wantErr: true},
		{name: "task without workflow", filters: workflowRunFilters{ConcurrencyTaskName: &taskName, ConcurrencyKey: &key}, wantErr: true},
		{name: "shared and workflow", filters: workflowRunFilters{ConcurrencyStrategyName: &sharedName, ConcurrencyWorkflowId: &workflowId, ConcurrencyKey: &key}, wantErr: true},
		{name: "empty shared name", filters: workflowRunFilters{ConcurrencyStrategyName: &empty, ConcurrencyKey: &key}, wantErr: true},
		{name: "invalid state", filters: workflowRunFilters{ConcurrencyWorkflowId: &workflowId, ConcurrencyKey: &key, ConcurrencySlotState: &badState}, wantErr: true},
	}

	// validation fails before the repository is used, so the service needs no config
	svc := &V1WorkflowRunsService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := svc.concurrencySlotExternalIds(context.Background(), uuid.New(), tt.filters)

			if tt.wantErr {
				require.ErrorIs(t, err, errInvalidConcurrencyFilter)
				return
			}

			require.NoError(t, err)
			require.Nil(t, ids)
		})
	}
}

func TestTaskExternalIds(t *testing.T) {
	runId, taskId := uuid.New(), uuid.New()

	require.Equal(t, []uuid.UUID{taskId}, *taskExternalIds(&v1.ConcurrencySlotExternalIds{
		RunExternalIds:  []uuid.UUID{runId},
		TaskExternalIds: []uuid.UUID{taskId},
	}))

	require.Equal(t, []uuid.UUID{runId}, *taskExternalIds(&v1.ConcurrencySlotExternalIds{
		RunExternalIds: []uuid.UUID{runId},
	}))
}
