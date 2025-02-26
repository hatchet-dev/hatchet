package scheduling

import (
	"fmt"
	"sort"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type WorkerStateManager struct {
	workers           map[string]*WorkerState
	workerStepWeights map[string][]WorkerWithWeight
	stepDesiredLabels map[string][]*dbsqlc.GetDesiredLabelsRow
}

func NewWorkerStateManager(
	slots []*Slot,
	workerLabels map[string][]*dbsqlc.GetWorkerLabelsRow,
	stepDesiredLabels map[string][]*dbsqlc.GetDesiredLabelsRow,
) *WorkerStateManager {

	workers := make(map[string]*WorkerState)
	workerStepWeights := make(map[string][]WorkerWithWeight, 0)

	// initialize worker states
	for _, slot := range slots {
		workerId := slot.WorkerId

		if _, ok := workers[workerId]; !ok {
			workers[workerId] = NewWorkerState(
				workerId,
				workerLabels[workerId],
			)
		}
		workers[workerId].AddSlot(slot)
	}

	// compute affinity weights
	for stepId, desired := range stepDesiredLabels {
		if len(desired) == 0 {
			continue
		}

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

func (wm *WorkerStateManager) AttemptAssignSlot(qi *QueueItemWithOrder) *Slot {

	// STICKY WORKERS
	if qi.Sticky.Valid {
		fmt.Println("STICKY WORKER")
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
	if workers, ok := wm.workerStepWeights[sqlchelpers.UUIDToStr(qi.StepId)]; ok && len(workers) > 0 {
		fmt.Println("AFFINITY WORKER", workers)
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
	workers := wm.getWorkersSortedBySlots()

	for _, worker := range workers {
		workerCp := worker
		slot := wm.attemptAssignToWorker(workerCp, qi)

		if slot == nil {
			continue
		}

		return slot
	}

	return nil
}

func (wm *WorkerStateManager) attemptAssignToWorker(worker *WorkerState, qi *QueueItemWithOrder) *Slot {
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

func (wm *WorkerStateManager) getWorkersSortedBySlots() []*WorkerState {
	workers := make([]*WorkerState, 0, len(wm.workers))

	for _, worker := range wm.workers {
		workers = append(workers, worker)
	}

	// sort the workers by the number of slots, descending
	sort.SliceStable(workers, func(i, j int) bool {
		return len(workers[i].slots) > len(workers[j].slots)
	})

	return workers
}
