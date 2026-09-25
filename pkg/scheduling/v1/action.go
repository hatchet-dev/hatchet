package v1

import (
	"maps"
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

	// starved is set on the run loop when an assignment for this action found
	// candidate workers but none with the free slots it requested. A heuristic
	// (non-forced) replenish rebuilds a starved action's pools regardless of
	// the active-slot thresholds, which count every slot type together and so
	// are blind to one type draining while another stays idle. Cleared when
	// the pools are rebuilt.
	starved *starvation
}

// starvation records the slot requests that starved assignments could not
// place, by the queue their items came from, so those queuers can be woken
// once a rebuild gives one of their requests room on a worker.
type starvation struct {
	requestsByQueue map[string][]map[string]int32
}

// markStarved runs on the run loop. It keeps the distinct requests per queue;
// a request already recorded for the queue is not added again.
func (a *action) markStarved(queue string, requests map[string]int32) {
	if a.starved == nil {
		a.starved = &starvation{
			requestsByQueue: make(map[string][]map[string]int32, 1),
		}
	}

	for _, seen := range a.starved.requestsByQueue[queue] {
		if maps.Equal(seen, requests) {
			return
		}
	}

	a.starved.requestsByQueue[queue] = append(a.starved.requestsByQueue[queue], maps.Clone(requests))
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
