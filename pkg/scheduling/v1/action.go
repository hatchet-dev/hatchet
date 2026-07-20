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
}

func (a *action) activeCountFromPools(poolsByWorker map[uuid.UUID]map[string]*slotPool, now time.Time) int {
	count := 0
	for _, workerId := range a.workerIds {
		for _, pool := range poolsByWorker[workerId] {
			if pool.staleAt(now) {
				return 0
			}
			count += pool.unusedCount()
		}
	}
	return count
}
