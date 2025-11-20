package v1

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func TestPartitionTasksByStepConfigMissingStep(t *testing.T) {
	stepA := uuid.NewString()
	stepB := uuid.NewString()

	tasks := []CreateTaskOpts{
		{StepId: stepA},
		{StepId: stepB},
	}

	stepConfig := map[string]*sqlcv1.ListStepsByIdsRow{
		stepA: {},
	}

	valid, missing := partitionTasksByStepConfig(tasks, stepConfig)
	require.Len(t, valid, 1)
	require.Equal(t, stepA, valid[0].StepId)
	require.Equal(t, []string{stepB}, missing)
}

func TestPartitionTasksByStepConfigAllPresent(t *testing.T) {
	stepA := uuid.NewString()
	stepB := uuid.NewString()

	tasks := []CreateTaskOpts{
		{StepId: stepA},
		{StepId: stepB},
	}

	stepConfig := map[string]*sqlcv1.ListStepsByIdsRow{
		stepA: {},
		stepB: {},
	}

	valid, missing := partitionTasksByStepConfig(tasks, stepConfig)
	require.Len(t, valid, 2)
	require.Empty(t, missing)
}

func TestPartitionTasksByStepConfigEmptyInput(t *testing.T) {
	valid, missing := partitionTasksByStepConfig(nil, nil)
	require.Nil(t, valid)
	require.Nil(t, missing)
}
