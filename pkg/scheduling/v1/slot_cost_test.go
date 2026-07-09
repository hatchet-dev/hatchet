//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// These tests use the scheduler's internal slot request map ({default: N}) directly, not the
// public SDK parameter.

func defaultSlots(w *worker, n int) []*slot {
	slots := make([]*slot, n)
	for i := range slots {
		slots[i] = newSlot(w, newSlotMeta([]string{"A"}, repo.SlotTypeDefault))
	}
	return slots
}

func TestSlotCost_DefaultCostFiveConsumesFiveSlots(t *testing.T) {
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	a, err := actionWithSlots("A", defaultSlots(w, 5)...)
	require.NoError(t, err)

	assigned := findAssignableSlots(a.slots, a, map[string]int32{repo.SlotTypeDefault: 5}, nil, nil)
	require.NotNil(t, assigned)
	require.Len(t, assigned.slots, 5)
	require.Equal(t, workerId, assigned.workerId())

	for _, sl := range assigned.slots {
		require.True(t, sl.isUsed())
	}

	none := findAssignableSlots(a.slots, a, map[string]int32{repo.SlotTypeDefault: 1}, nil, nil)
	require.Nil(t, none)
}

func TestSlotCost_MixedHeavyAndLightShareDefaultPool(t *testing.T) {
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	a, err := actionWithSlots("A", defaultSlots(w, 6)...)
	require.NoError(t, err)

	heavy := findAssignableSlots(a.slots, a, map[string]int32{repo.SlotTypeDefault: 5}, nil, nil)
	require.NotNil(t, heavy)
	require.Len(t, heavy.slots, 5)

	light := findAssignableSlots(a.slots, a, map[string]int32{repo.SlotTypeDefault: 1}, nil, nil)
	require.NotNil(t, light)
	require.Len(t, light.slots, 1)

	none := findAssignableSlots(a.slots, a, map[string]int32{repo.SlotTypeDefault: 1}, nil, nil)
	require.Nil(t, none)
}

func TestSlotCost_ReservationMustFitOnOneWorker(t *testing.T) {
	w1 := &worker{ListActiveWorkersResult: testWorker(uuid.New())}
	w2 := &worker{ListActiveWorkersResult: testWorker(uuid.New())}

	all := append(defaultSlots(w1, 4), defaultSlots(w2, 4)...)
	a, err := actionWithSlots("A", all...)
	require.NoError(t, err)

	none := findAssignableSlots(a.slots, a, map[string]int32{repo.SlotTypeDefault: 5}, nil, nil)
	require.Nil(t, none)

	fits := findAssignableSlots(a.slots, a, map[string]int32{repo.SlotTypeDefault: 4}, nil, nil)
	require.NotNil(t, fits)
	require.Len(t, fits.slots, 4)
}

func TestSlotCost_CostExceedingCapacityRemainsUnscheduled(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	a, err := actionWithSlots("A", defaultSlots(w, 4)...)
	require.NoError(t, err)

	qi := testQI(tenantId, "A", 1)
	req := map[string]int32{repo.SlotTypeDefault: 5}

	res, err := s.tryAssignSingleton(context.Background(), qi, a, a.slots, 0, nil, req, func() {}, func() {})
	require.NoError(t, err)
	require.False(t, res.succeeded)
	require.True(t, res.noSlots)
}

func TestSlotCost_NoExplicitRequestConsumesOneSlotPerTask(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	a, err := actionWithSlots("A", defaultSlots(w, 2)...)
	require.NoError(t, err)
	s.actions["A"] = a

	qis := []*sqlcv1.V1QueueItem{
		testQI(tenantId, "A", 1),
		testQI(tenantId, "A", 2),
	}

	// With no entry in stepIdsToRequests, the scheduler falls back to {default: 1}.
	res, _, err := s.tryAssignBatch(context.Background(), "A", qis, 0,
		map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow{}, map[uuid.UUID]map[string]int32{}, nil, nil)
	require.NoError(t, err)

	assigned := 0
	for _, r := range res {
		if r.succeeded {
			assigned++
		}
	}
	require.Equal(t, 2, assigned)
}

func TestSlotCost_ExplicitDefaultCostBlocksProportionally(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	a, err := actionWithSlots("A", defaultSlots(w, 2)...)
	require.NoError(t, err)
	s.actions["A"] = a

	qi1 := testQI(tenantId, "A", 1)
	qi2 := testQI(tenantId, "A", 2)
	qis := []*sqlcv1.V1QueueItem{qi1, qi2}

	stepRequests := map[uuid.UUID]map[string]int32{
		qi1.StepID: {repo.SlotTypeDefault: 2},
		qi2.StepID: {repo.SlotTypeDefault: 2},
	}

	res, _, err := s.tryAssignBatch(context.Background(), "A", qis, 0,
		map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow{}, stepRequests, nil, nil)
	require.NoError(t, err)

	assigned, noSlots := 0, 0
	for _, r := range res {
		if r.succeeded {
			assigned++
		}
		if r.noSlots {
			noSlots++
		}
	}
	require.Equal(t, 1, assigned)
	require.Equal(t, 1, noSlots)
}
