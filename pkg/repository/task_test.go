package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestPartitionInsertTasksByStepConfigMissingStep(t *testing.T) {
	stepA := uuid.New()
	stepB := uuid.New()

	tasks := []CreateTaskOpts{
		{StepId: stepA},
		{StepId: stepB},
	}

	stepConfig := map[uuid.UUID]*sqlcv1.ListStepsByIdsRow{
		stepA: {},
	}

	valid, missing := partitionInsertTasksByStepConfig(tasks, stepConfig)
	require.Len(t, valid, 1)
	require.Equal(t, stepA, valid[0].StepId)
	require.Equal(t, []uuid.UUID{stepB}, missing)
}

func TestPartitionInsertTasksByStepConfigAllPresent(t *testing.T) {
	stepA := uuid.New()
	stepB := uuid.New()

	tasks := []CreateTaskOpts{
		{StepId: stepA},
		{StepId: stepB},
	}

	stepConfig := map[uuid.UUID]*sqlcv1.ListStepsByIdsRow{
		stepA: {},
		stepB: {},
	}

	valid, missing := partitionInsertTasksByStepConfig(tasks, stepConfig)
	require.Len(t, valid, 2)
	require.Empty(t, missing)
}

func TestPartitionInsertTasksByStepConfigEmptyInput(t *testing.T) {
	valid, missing := partitionInsertTasksByStepConfig(nil, nil)
	require.Nil(t, valid)
	require.Nil(t, missing)
}

func TestPartitionReplayTasksByStepConfigMissingStep(t *testing.T) {
	stepA := uuid.New()
	stepB := uuid.New()

	tasks := []ReplayTaskOpts{
		{StepId: stepA},
		{StepId: stepB},
	}

	stepConfig := map[uuid.UUID]*sqlcv1.ListStepsByIdsRow{
		stepA: {},
	}

	valid, missing := partitionReplayTasksByStepConfig(tasks, stepConfig)
	require.Len(t, valid, 1)
	require.Equal(t, stepA, valid[0].StepId)
	require.Equal(t, []uuid.UUID{stepB}, missing)
}

func TestPartitionReplayTasksByStepConfigAllPresent(t *testing.T) {
	stepA := uuid.New()
	stepB := uuid.New()

	tasks := []ReplayTaskOpts{
		{StepId: stepA},
		{StepId: stepB},
	}

	stepConfig := map[uuid.UUID]*sqlcv1.ListStepsByIdsRow{
		stepA: {},
		stepB: {},
	}

	valid, missing := partitionReplayTasksByStepConfig(tasks, stepConfig)
	require.Len(t, valid, 2)
	require.Empty(t, missing)
}

func TestPartitionReplayTasksByStepConfigEmptyInput(t *testing.T) {
	valid, missing := partitionReplayTasksByStepConfig(nil, nil)
	require.Nil(t, valid)
	require.Nil(t, missing)
}
