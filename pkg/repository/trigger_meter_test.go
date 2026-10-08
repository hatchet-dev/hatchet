//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

var errStopAfterMeter = errors.New("stop after meter")

type recordingMeter struct {
	TenantLimitRepository
	charges []int32
}

func (m *recordingMeter) Meter(_ context.Context, _ sqlcv1.DBTX, resource sqlcv1.LimitResource, _ uuid.UUID, n int32) (func() error, func()) {
	if resource == sqlcv1.LimitResourceTASKRUN {
		m.charges = append(m.charges, n)
	}

	return func() error { return errStopAfterMeter }, func() {}
}

func dagSteps(workflowVersionId uuid.UUID, userSteps int, withOnFailure, withOrchestrator bool) []*sqlcv1.ListStepsByWorkflowVersionIdsRow {
	steps := make([]*sqlcv1.ListStepsByWorkflowVersionIdsRow, 0, userSteps+2)

	var prev *uuid.UUID

	for i := 0; i < userSteps; i++ {
		row := &sqlcv1.ListStepsByWorkflowVersionIdsRow{
			ID:                uuid.New(),
			ActionId:          fmt.Sprintf("wf:step-%d", i),
			WorkflowVersionId: workflowVersionId,
			JobKind:           sqlcv1.JobKindDEFAULT,
		}

		if prev != nil {
			row.Parents = []uuid.UUID{*prev}
		}

		id := row.ID
		prev = &id
		steps = append(steps, row)
	}

	if withOnFailure {
		steps = append(steps, &sqlcv1.ListStepsByWorkflowVersionIdsRow{
			ID:                uuid.New(),
			ActionId:          "wf:on-failure",
			WorkflowVersionId: workflowVersionId,
			JobKind:           sqlcv1.JobKindONFAILURE,
		})
	}

	if withOrchestrator {
		steps = append(steps, &sqlcv1.ListStepsByWorkflowVersionIdsRow{
			ID:                uuid.New(),
			ActionId:          "wf:orchestrator",
			WorkflowVersionId: workflowVersionId,
			JobKind:           sqlcv1.JobKindDEFAULT,
			IsDurable:         true,
			IsDagOrchestrator: true,
		})
	}

	return steps
}

func newMeterTestRepo(meter *recordingMeter, workflowVersionId uuid.UUID, steps []*sqlcv1.ListStepsByWorkflowVersionIdsRow) *sharedRepository {
	l := zerolog.Nop()
	cache := expirable.NewLRU(10, func(uuid.UUID, []*sqlcv1.ListStepsByWorkflowVersionIdsRow) {}, time.Hour)
	cache.Add(workflowVersionId, steps)

	return &sharedRepository{
		l:                           &l,
		m:                           meter,
		stepsInWorkflowVersionCache: cache,
	}
}

// meterChargesForRun returns the TASK_RUN charges for one run: the whole-workflow trigger, plus,
// when the workflow has an orchestrator, one targeted trigger per step the DAG operator spawns.
func meterChargesForRun(t *testing.T, steps []*sqlcv1.ListStepsByWorkflowVersionIdsRow, workflowVersionId uuid.UUID) []int32 {
	t.Helper()

	meter := &recordingMeter{}
	repo := newMeterTestRepo(meter, workflowVersionId, steps)
	tenantId := uuid.New()

	trigger := func(targetActionId *string) {
		_, _, _, _, _, _, err := repo.triggerWorkflowsCore(context.Background(), &OptimisticTx{}, tenantId, []triggerTuple{{
			externalId:        uuid.New(),
			workflowVersionId: workflowVersionId,
			targetActionId:    targetActionId,
		}}, nil, false)
		require.ErrorIs(t, err, errStopAfterMeter)
	}

	trigger(nil)

	for _, s := range steps {
		if s.IsDagOrchestrator || !hasOrchestrator(steps) {
			continue
		}

		actionId := s.ActionId
		trigger(&actionId)
	}

	return meter.charges
}

func hasOrchestrator(steps []*sqlcv1.ListStepsByWorkflowVersionIdsRow) bool {
	for _, s := range steps {
		if s.IsDagOrchestrator {
			return true
		}
	}

	return false
}

func TestTaskRunMeter_OperatorDagChargesUserStepsOnce(t *testing.T) {
	wv := uuid.New()

	require.Equal(t, []int32{3, 0, 0, 0}, meterChargesForRun(t, dagSteps(wv, 3, false, true), wv))
}

func TestTaskRunMeter_OperatorDagWithOnFailureStep(t *testing.T) {
	wv := uuid.New()

	require.Equal(t, []int32{4, 0, 0, 0, 0}, meterChargesForRun(t, dagSteps(wv, 3, true, true), wv))
}

func TestTaskRunMeter_OperatorDagIsLinearInSteps(t *testing.T) {
	wv := uuid.New()

	total := int32(0)
	for _, c := range meterChargesForRun(t, dagSteps(wv, 28, false, true), wv) {
		total += c
	}

	require.Equal(t, int32(28), total)
}

func TestTaskRunMeter_DagWithoutOperatorChargesAllSteps(t *testing.T) {
	wv := uuid.New()

	require.Equal(t, []int32{4}, meterChargesForRun(t, dagSteps(wv, 3, true, false), wv))
}

func TestTaskRunMeter_SingleTask(t *testing.T) {
	wv := uuid.New()

	require.Equal(t, []int32{1}, meterChargesForRun(t, dagSteps(wv, 1, false, false), wv))
}

func TestTenantLimitMeter_ZeroChargeSkipsLimitCheck(t *testing.T) {
	// nil cache and queries: any limit lookup would panic
	repo := &tenantLimitRepository{enforceLimits: true}

	precommit, _ := repo.Meter(context.Background(), nil, sqlcv1.LimitResourceTASKRUN, uuid.New(), 0)

	require.NoError(t, precommit())
}
