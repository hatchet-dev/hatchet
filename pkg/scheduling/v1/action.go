package v1

import (
	"sort"
	"strings"
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

	// starved records, per queue, the slot-type sets its missed items asked for
	// (types with units > 0) as slotTypeSetKeys. One entry is one starvation
	// episode: the first miss on a (queue, type set) requests a replenish and
	// later misses on the same entry request nothing. Units are not kept on
	// purpose: a request for more units than any worker has free is a
	// constrained item, not starvation.
	starved map[string]map[slotTypeSetKey]struct{}
}

// slotTypeSetKey is the canonical encoding of the slot types a request asks
// for: the one type itself for a single-type request, the sorted types joined
// by slotTypeSeparator otherwise. Slot types are Postgres text values, which
// cannot hold NUL, so no type contains the separator.
type slotTypeSetKey string

const slotTypeSeparator = "\x00"

// slotTypeSetOf returns the key of the types in request with units > 0 and
// reports whether there is at least one. It does not allocate for a
// single-type request.
func slotTypeSetOf(request map[string]int32) (slotTypeSetKey, bool) {
	var single string
	n := 0

	for slotType, units := range request {
		if units > 0 {
			single = slotType
			n++
		}
	}

	switch n {
	case 0:
		return "", false
	case 1:
		return slotTypeSetKey(single), true
	}

	types := make([]string, 0, n)
	for slotType, units := range request {
		if units > 0 {
			types = append(types, slotType)
		}
	}
	sort.Strings(types)

	return slotTypeSetKey(strings.Join(types, slotTypeSeparator)), true
}

// types decodes the key back into its sorted slot types.
func (k slotTypeSetKey) types() []string {
	return strings.Split(string(k), slotTypeSeparator)
}

// IsStarved reports whether any entry is recorded on this action. Runs on the
// run loop.
func (a *action) IsStarved() bool {
	return len(a.starved) > 0
}

// MarkStarved records that an item from queue missed on this action with the
// given slot request and reports whether that (queue, type set) entry is new,
// so the caller can act on that transition alone: a repeated miss on an
// existing entry reports false, and a miss on a type set the queue has not
// recorded reports true even when the action is starved for other sets. Slot
// types with no units are not requested and so not recorded; a request with
// none records nothing. Runs on the run loop.
func (a *action) MarkStarved(queue string, request map[string]int32) (newlyStarved bool) {
	key, ok := slotTypeSetOf(request)
	if !ok {
		return false
	}

	entries := a.starved[queue]
	if _, exists := entries[key]; exists {
		return false
	}

	if entries == nil {
		if a.starved == nil {
			a.starved = make(map[string]map[slotTypeSetKey]struct{}, 1)
		}

		entries = make(map[slotTypeSetKey]struct{}, 1)
		a.starved[queue] = entries
	}

	entries[key] = struct{}{}

	return true
}

// RestoreStarved clears every entry for which some worker's rebuilt pools have
// a free slot of each type in the entry's set, and returns the queues that had
// at least one entry cleared. A queue keeps its other entries, so the item
// whose type set is still drained does not request another replenish when it
// misses again, and the next rebuild that frees its types restores it. The
// check is per worker, not per type: types free on different workers restore
// nothing, since the woken item would miss again and re-mark. Runs on the run
// loop.
func (a *action) RestoreStarved(poolsByWorker map[uuid.UUID]map[string]*slotPool, workerIds []uuid.UUID, now time.Time) (restoredQueues []string) {
	for queue, entries := range a.starved {
		restored := false

		for key := range entries {
			if hasFreeSlotOfEachTypeIn(poolsByWorker, workerIds, key.types(), now) {
				delete(entries, key)
				restored = true
			}
		}

		if len(entries) == 0 {
			delete(a.starved, queue)
		}

		if restored {
			restoredQueues = append(restoredQueues, queue)
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

// hasFreeSlotOfEachTypeIn is hasFreeSlotOfEachType over a recorded type set.
func hasFreeSlotOfEachTypeIn(poolsByWorker map[uuid.UUID]map[string]*slotPool, workerIds []uuid.UUID, slotTypes []string, now time.Time) bool {
	for _, workerId := range workerIds {
		pools := poolsByWorker[workerId]
		free := true

		for _, slotType := range slotTypes {
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
