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
