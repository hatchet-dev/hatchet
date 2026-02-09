package v1

import (
	"slices"
	"sync"

	"github.com/google/uuid"
)

type action struct {
	mu       sync.RWMutex
	actionId string

	lastReplenishedSlotCount   int
	lastReplenishedWorkerCount int

	// note that slots can be used across multiple actions, hence the pointer
	slots []*slot

	// slotsByTypeAndWorkerId indexes slots by slotType -> workerId -> slots.
	//
	// NOTE: this index contains pointers to the same slot objects as slots.
	// It is built/replaced during replenish under a.mu and read during assignment
	// under a.mu (RLock or Lock).
	slotsByTypeAndWorkerId map[string]map[uuid.UUID][]*slot
}

func (a *action) activeCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	count := 0

	for _, slot := range a.slots {
		if slot.active() {
			count++
		}
	}

	return count
}

// orderedLock acquires the locks in a stable order to prevent deadlocks
func orderedLock(actionsMap map[string]*action) {
	actions := sortActions(actionsMap)

	for _, action := range actions {
		action.mu.Lock()
	}
}

// orderedUnlock releases the locks in a stable order to prevent deadlocks. it returns
// a function that should be deferred to unlock the locks.
func orderedUnlock(actionsMap map[string]*action) func() {
	actions := sortActions(actionsMap)

	return func() {
		for _, action := range actions {
			action.mu.Unlock()
		}
	}
}

func sortActions(actionsMap map[string]*action) []*action {
	actions := make([]*action, 0, len(actionsMap))

	for _, action := range actionsMap {
		actions = append(actions, action)
	}

	slices.SortStableFunc(actions, func(i, j *action) int {
		switch {
		case i.actionId < j.actionId:
			return -1
		case i.actionId > j.actionId:
			return 1
		default:
			return 0
		}
	})

	return actions
}
