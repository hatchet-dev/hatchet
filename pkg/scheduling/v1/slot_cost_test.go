//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"testing"
	"time"

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

func TestSlotCost_MixedHeavyAndLightShareDefaultPool(t *testing.T) {
	s := newTestScheduler(t, uuid.New(), &mockAssignmentRepo{})
	workerId := uuid.New()
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	a := seedActionPools(t, s, "A", defaultSlots(w, 6)...)

	heavy := s.findAssignableWorkerPools(a.workerIds, map[string]int32{repo.SlotTypeDefault: 5}, nil, nil)
	require.NotNil(t, heavy)
	require.Len(t, heavy.slots, 5)

	light := s.findAssignableWorkerPools(a.workerIds, map[string]int32{repo.SlotTypeDefault: 1}, nil, nil)
	require.NotNil(t, light)
	require.Len(t, light.slots, 1)

	none := s.findAssignableWorkerPools(a.workerIds, map[string]int32{repo.SlotTypeDefault: 1}, nil, nil)
	require.Nil(t, none)
}

func TestSlotCost_ReservationMustFitOnOneWorker(t *testing.T) {
	s := newTestScheduler(t, uuid.New(), &mockAssignmentRepo{})
	w1 := &worker{ListActiveWorkersResult: testWorker(uuid.New())}
	w2 := &worker{ListActiveWorkersResult: testWorker(uuid.New())}

	all := append(defaultSlots(w1, 4), defaultSlots(w2, 4)...)
	a := seedActionPools(t, s, "A", all...)

	none := s.findAssignableWorkerPools(a.workerIds, map[string]int32{repo.SlotTypeDefault: 5}, nil, nil)
	require.Nil(t, none)

	fits := s.findAssignableWorkerPools(a.workerIds, map[string]int32{repo.SlotTypeDefault: 4}, nil, nil)
	require.NotNil(t, fits)
	require.Len(t, fits.slots, 4)
}

// An over-capacity task is unassigned only while inside its schedule timeout. Past the timeout the
// scheduler routes it to schedulingTimedOut, which the engine cancels with reason
// SCHEDULING_TIMED_OUT, so the wait is bounded.
func TestSlotCost_OverCapacityWaitsThenSchedulingTimesOut(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	seedActionPools(t, s, "A", defaultSlots(w, 4)...)

	waiting := testQI(tenantId, "A", 1)
	waiting.ScheduleTimeoutAt = ts(time.Now().UTC().Add(5 * time.Minute))

	expired := testQI(tenantId, "A", 2)
	expired.ScheduleTimeoutAt = ts(time.Now().UTC().Add(-1 * time.Second))

	stepRequests := map[uuid.UUID]map[string]int32{
		waiting.StepID: {repo.SlotTypeDefault: 5},
		expired.StepID: {repo.SlotTypeDefault: 5},
	}

	ch := s.tryAssign(
		context.Background(),
		[]*sqlcv1.V1QueueItem{waiting, expired},
		map[uuid.UUID][]*sqlcv1.GetDesiredLabelsRow{},
		stepRequests,
		nil,
		nil,
	)

	assigned := map[int64]bool{}
	unassigned := map[int64]bool{}
	timedOut := map[int64]bool{}

	for r := range ch {
		for _, as := range r.assigned {
			assigned[as.QueueItem.TaskID] = true
		}
		for _, u := range r.unassigned {
			unassigned[u.TaskID] = true
		}
		for _, to := range r.schedulingTimedOut {
			timedOut[to.TaskID] = true
		}
	}

	require.Empty(t, assigned)
	require.True(t, unassigned[waiting.TaskID])
	require.False(t, timedOut[waiting.TaskID])
	require.True(t, timedOut[expired.TaskID])
}

func TestSlotCost_ExplicitDefaultCostBlocksProportionally(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()

	s := newTestScheduler(t, tenantId, &mockAssignmentRepo{})
	w := &worker{ListActiveWorkersResult: testWorker(workerId)}

	seedActionPools(t, s, "A", defaultSlots(w, 2)...)

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
