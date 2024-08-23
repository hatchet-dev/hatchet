package scheduling

import (
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type WorkerStateManager struct {
	workers           map[string]*WorkerState
	workerStepWeights map[string][]WorkerWithWeight
	stepDesiredLabels map[string][]*dbsqlc.GetDesiredLabelsRow
}

func NewWorkerStateManager(
	slots []*dbsqlc.ListSemaphoreSlotsToAssignRow,
	workerLabels map[string][]*dbsqlc.GetWorkerLabelsRow,
	stepDesiredLabels map[string][]*dbsqlc.GetDesiredLabelsRow,
) *WorkerStateManager {

	workers := make(map[string]*WorkerState)
	workerStepWeights := make(map[string][]WorkerWithWeight, 0)

	// initialize worker states
	for _, slot := range slots {
		workerId := sqlchelpers.UUIDToStr(slot.WorkerId)

		if _, ok := workers[workerId]; !ok {
			workers[workerId] = NewWorkerState(
				workerId,
				workerLabels[workerId],
			)
		}
		workers[sqlchelpers.UUIDToStr(slot.WorkerId)].AddSlot(slot)
	}

	// compute affinity weights
	for stepId, desired := range stepDesiredLabels {
		for workerId, worker := range workers {
			weight := ComputeWeight(desired, worker.labels)

			// cache the weight on the worker
			workers[workerId].AddStepWeight(stepId, weight)

			workerStepWeights[stepId] = append(workerStepWeights[stepId], WorkerWithWeight{
				WorkerId: workerId,
				Weight:   weight,
			})
		}
	}

	// sort the weights
	for _, weights := range workerStepWeights {
		SortWorkerWeights(weights)
	}

	return &WorkerStateManager{
		workers:           workers,
		workerStepWeights: workerStepWeights,
		stepDesiredLabels: stepDesiredLabels,
	}
}

func (wm *WorkerStateManager) HasEligibleWorkers(stepId string) bool {
	// desired labels
	if weightedWorkers, ok := wm.workerStepWeights[stepId]; ok {
		return len(weightedWorkers) > 0
	}

	return len(wm.workers) > 0
}

func (wm *WorkerStateManager) AttemptAssignSlot(qi *QueueItemWithOrder) *dbsqlc.ListSemaphoreSlotsToAssignRow {

	// STICKY WORKERS
	if qi.Sticky.Valid {
		if worker, ok := wm.workers[sqlchelpers.UUIDToStr(qi.DesiredWorkerId)]; ok {
			slot := wm.attemptAssignToWorker(worker, qi)

			if slot != nil {
				return slot
			}
			return nil
		}

		if qi.DesiredWorkerId.Valid && qi.Sticky.StickyStrategy == dbsqlc.StickyStrategyHARD {
			// if we have a HARD sticky worker and we can't find it then return nil
			// to indicate that we can't assign the slot
			return nil
		}
	} // if we reached this with sticky we'll try to find an alternative worker

	// AFFINITY WORKERS
	if workers, ok := wm.workerStepWeights[sqlchelpers.UUIDToStr(qi.StepId)]; ok {
		for _, workerWW := range workers {

			worker := wm.workers[workerWW.WorkerId]

			if worker == nil {
				// the worker is probably exhausted we should consider removing from the workerWW list
				continue
			}

			slot := wm.attemptAssignToWorker(worker, qi)

			if slot == nil {
				continue
			}

			return slot
		}

		return nil
	}

	// DEFAULT STRATEGY
	workers := wm.workers
	for _, worker := range workers {

		slot := wm.attemptAssignToWorker(worker, qi)

		if slot == nil {
			continue
		}

		return slot
	}

	return nil
}

func (wm *WorkerStateManager) attemptAssignToWorker(worker *WorkerState, qi *QueueItemWithOrder) *dbsqlc.ListSemaphoreSlotsToAssignRow {
	slot, isEmpty := worker.AssignSlot(qi)

	if slot == nil {
		// if we can't assign the slot to the worker then continue
		return nil
	}

	// cleanup the worker if it's empty
	if isEmpty {
		wm.DropWorker(worker.workerId)
	}

	// finally, return the slot
	return slot
}

func (wm *WorkerStateManager) DropWorker(workerId string) {
	// delete the worker
	delete(wm.workers, workerId)

	// cleanup the step weights
	// TODO
}
