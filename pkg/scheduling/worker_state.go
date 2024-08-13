package scheduling

import (
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type WorkerState struct {
	workerId    string
	slots       map[string]*dbsqlc.ListSemaphoreSlotsToAssignRow
	actionIds   map[string]struct{}
	labels      []*dbsqlc.GetWorkerLabelsRow
	stepWeights map[string]int
}

func NewWorkerState(workerId string, labels []*dbsqlc.GetWorkerLabelsRow) *WorkerState {
	return &WorkerState{
		workerId:    workerId,
		slots:       make(map[string]*dbsqlc.ListSemaphoreSlotsToAssignRow),
		actionIds:   make(map[string]struct{}),
		labels:      labels,
		stepWeights: make(map[string]int),
	}
}

func (w *WorkerState) AddStepWeight(stepId string, weight int) {
	w.stepWeights[stepId] = weight
}

func (w *WorkerState) AddSlot(slot *dbsqlc.ListSemaphoreSlotsToAssignRow) {
	w.slots[sqlchelpers.UUIDToStr(slot.ID)] = slot
	w.actionIds[slot.ActionId] = struct{}{}
}

func (w *WorkerState) CanAssign(action string, stepId *string) bool {
	if _, ok := w.actionIds[action]; !ok {
		return false
	}

	if stepId == nil {
		return true
	}

	if weight, ok := w.stepWeights[*stepId]; ok {
		return weight >= 0
	}

	return true
}

func (w *WorkerState) AssignSlot(qi *QueueItemWithOrder) (*dbsqlc.ListSemaphoreSlotsToAssignRow, bool) {

	// if the actionId is not in the worker's actionIds, then we can't assign this slot
	stepId := sqlchelpers.UUIDToStr(qi.StepId)
	if !w.CanAssign(qi.ActionId.String, &stepId) {
		return nil, false
	}

	// pop the first slot
	slot := w.popRandomSlot(w.slots)
	isEmpty := len(w.slots) == 0

	return slot, isEmpty
}

func (w *WorkerState) popRandomSlot(slots map[string]*dbsqlc.ListSemaphoreSlotsToAssignRow) *dbsqlc.ListSemaphoreSlotsToAssignRow {
	for id, slot := range slots {
		delete(slots, id)
		return slot
	}

	return nil
}
