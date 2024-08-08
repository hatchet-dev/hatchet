package scheduling

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type WorkerState struct {
	workerId  string
	slots     []*dbsqlc.ListSemaphoreSlotsToAssignRow
	actionIds map[string]struct{}
	labels    []*dbsqlc.GetWorkerLabelsRow
}

func NewWorkerState(workerId string, labels []*dbsqlc.GetWorkerLabelsRow) *WorkerState {
	return &WorkerState{
		workerId:  workerId,
		slots:     make([]*dbsqlc.ListSemaphoreSlotsToAssignRow, 0),
		actionIds: make(map[string]struct{}),
		labels:    labels,
	}
}

func (w *WorkerState) AddSlot(slot *dbsqlc.ListSemaphoreSlotsToAssignRow) {
	w.slots = append(w.slots, slot)

	if _, ok := w.actionIds[slot.ActionId]; !ok {
		w.actionIds[slot.ActionId] = struct{}{}
	}
}

func (w *WorkerState) CanAssign(qi *QueueItemWithOrder, desiredLabels []*dbsqlc.GetDesiredLabelsRow) bool {
	if _, ok := w.actionIds[qi.ActionId.String]; !ok {
		return false
	}

	if len(desiredLabels) > 0 {

		// TODO cache
		weight := ComputeWeight(desiredLabels, w.labels)

		fmt.Println(weight)
		return weight >= 0
	}

	return true
}

func (w *WorkerState) AssignSlot(qi *QueueItemWithOrder, desiredLabels []*dbsqlc.GetDesiredLabelsRow) (*dbsqlc.ListSemaphoreSlotsToAssignRow, bool) {

	// if the actionId is not in the worker's actionIds, then we can't assign this slot
	if !w.CanAssign(qi, desiredLabels) {
		return nil, false
	}

	// pop the first slot
	slot := w.slots[0]
	w.slots = w.slots[1:]

	isEmpty := len(w.slots) == 0

	return slot, isEmpty
}
