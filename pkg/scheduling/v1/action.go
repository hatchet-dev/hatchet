package v1

import (
	"time"

	"github.com/google/uuid"
)

type action struct {
	lastReplenishedSlotCount   int
	lastReplenishedWorkerCount int

	// workerIds is the thin action index into Scheduler.pools.
	workerIds []uuid.UUID

	// ringOffset rotates the starting worker between assignments so load spreads
	// across the action's workers. Sticky and label ranking keep higher scores
	// first; the offset only rotates within the highest-rank tied group.
	ringOffset int

	// stores a list of starved queues (queues for this action which have 0 slots)
	starved map[string]map[string]struct{}
}

// IsStarved reports whether any queue is recorded on this action. Runs on the
// run loop.
func (a *action) IsStarved() bool {
	return len(a.starved) > 0
}

// MarkStarved records that an item from queue missed on this action with the
// given slot request and reports whether the action was not starved before,
// so the caller can act on that transition alone. Slot types with no units are
// not requested and so not recorded; a request with none records nothing. Runs
// on the run loop.
func (a *action) MarkStarved(queue string, request map[string]int32) (newlyStarved bool) {
	types := a.starved[queue]

	for slotType, units := range request {
		if units <= 0 {
			continue
		}

		if types == nil {
			if a.starved == nil {
				a.starved = make(map[string]map[string]struct{}, 1)
			}

			newlyStarved = len(a.starved) == 0
			types = make(map[string]struct{}, len(request))
			a.starved[queue] = types
		}

		types[slotType] = struct{}{}
	}

	return newlyStarved
}

// RestoreStarved clears and returns the starved queues for which some worker's
// rebuilt pools have a free slot of every recorded slot type. A queue that no
// worker can serve stays marked, so its repeated misses do not request another
// replenish and the next rebuild that frees the slot restores it. The check is
// per worker, not per type: types free on different workers restore nothing,
// since the woken item would miss again and re-mark. Runs on the run loop.
func (a *action) RestoreStarved(poolsByWorker map[uuid.UUID]map[string]*slotPool, workerIds []uuid.UUID, now time.Time) (restoredQueues []string) {
	for queue, types := range a.starved {
		if hasFreeSlotOfEachType(poolsByWorker, workerIds, types, now) {
			restoredQueues = append(restoredQueues, queue)
			delete(a.starved, queue)
		}
	}

	return restoredQueues
}

// hasFreeSlotOfEachType reports whether one of the workers has at least one
// free slot of every given type. It is the starvation predicate: a miss marks
// when it is false over the live pools and a rebuild restores when it is true
// over the rebuilt ones. Units are ignored on purpose (see action.starved).
func hasFreeSlotOfEachType[V any](poolsByWorker map[uuid.UUID]map[string]*slotPool, workerIds []uuid.UUID, slotTypes map[string]V, now time.Time) bool {
	for _, workerId := range workerIds {
		pools := poolsByWorker[workerId]
		free := true

		for slotType := range slotTypes {
			if pools[slotType].freeCountAt(now) == 0 {
				free = false
				break
			}
		}

		if free {
			return true
		}
	}

	return false
}

func (a *action) activeCount(poolsByWorker map[uuid.UUID]map[string]*slotPool, now time.Time) int {
	count := 0
	for _, workerId := range a.workerIds {
		for _, pool := range poolsByWorker[workerId] {
			if pool.staleAt(now) {
				return 0
			}
			count += len(pool.free)
		}
	}
	return count
}
