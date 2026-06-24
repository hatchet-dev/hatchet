package concurrency

import (
	"cmp"
	"time"
)

// slotIndex is a data structure for efficiently querying concurrency slots.
//
// the basic idea here is: the queued runs could potentially be huge, and there are lots of strategies
// to store this type of index (local disk, compression, database, etc) all with different tradeoffs.
// i'd like to make it super easy to swap out this index.
type slotIndex interface {
	// write operations
	insert(slot)
	delete(taskId int64) (slot, bool)
	pop(n int) []slot
	popTimedOut(now time.Time) []slot

	// read operations
	get(taskId int64) (slot, bool)

	// peek returns the slot at the head of the index (the smallest under its comparator) without
	// removing it; ok is false if the index is empty. The cancel strategies use it to inspect the
	// best queued slot and the worst running slot (the running index is built with the reversed
	// comparator) while merging, so they never snapshot-and-sort.
	peek() (slot, bool)

	len() int

	// undo log: begin opens a scope in which write operations are recorded; commit discards the log
	// (writes are durable downstream), rollback reverses every recorded write to restore the index to
	// its state at begin().
	begin()
	commit()
	rollback()
}

// timeoutEntry is the timeout queue's element: just the key (taskId) and the schedule timeout it is
// ordered by. The full slot lives in the priority queue; popTimedOut fetches it from there, so the
// timeout heap stays small (16 bytes/entry vs a full slot copy).
type timeoutEntry struct {
	taskId      int64
	timeoutAtMs int64
}

// slotLocation tracks a task's position in both heaps in a single map entry. Indices are stored
// 1-based so the zero value means "absent from that heap" - this disambiguates a genuine index 0
// from a missing entry, and lets us drop the map entry exactly when the task leaves both heaps.
type slotLocation struct {
	priIdx int // 1-based index into priorityQueue.values; 0 = absent
	toIdx  int // 1-based index into timedOutQueue.values; 0 = absent (always 0 when timeouts untracked)
}

// inMemorySlotIndex is an implementation of the slotIndex interface using a indexed heap and a map for
// quick lookups.
// important: inMemorySlotIndex is NOT concurrency safe; callers must ensure appropriate synchronization.
type inMemorySlotIndex struct {
	priorityQueue *heap[slot]

	// note: this can be nil, for example if the index doesn't need to support scheduling timeouts
	timedOutQueue *heap[timeoutEntry] // separate heap ordered by schedule timeout for efficient timeout queries

	// byTaskID maps a taskId to its position in each heap. Shared across both heaps to avoid a second
	// map's key storage and bucket overhead.
	byTaskID map[int64]slotLocation
}

// priorityCompare is the default slot ordering, shared by GROUP_ROUND_ROBIN and CANCEL_IN_PROGRESS:
// higher priority first, then earlier taskInsertedAt, then lower taskId. It is a total order
// (taskId is unique per slot), so it is tie-free. Cancel strategies that keep the opposite end (e.g.
// CANCEL_NEWEST) pass reverseCompare(priorityCompare).
func priorityCompare(a, b slot) int {
	if a.priority != b.priority {
		return cmp.Compare(b.priority, a.priority)
	}
	if a.taskInsertedAtNs != b.taskInsertedAtNs {
		return cmp.Compare(a.taskInsertedAtNs, b.taskInsertedAtNs)
	}
	return cmp.Compare(a.taskId, b.taskId)
}

// reverseCompare flips a comparator's order while preserving ties, so an index built with it pops
// the worst element (under the original order) first.
func reverseCompare(c func(a, b slot) int) func(a, b slot) int {
	return func(a, b slot) int { return c(b, a) }
}

// newInMemorySlotIndex builds an index ordered by priorityCompare. Use newInMemorySlotIndexWithCompare
// to supply a different ordering (e.g. the reversed comparator for a running index).
func newInMemorySlotIndex(trackTimeouts bool) *inMemorySlotIndex {
	return newInMemorySlotIndexWithCompare(trackTimeouts, priorityCompare)
}

func newInMemorySlotIndexWithCompare(trackTimeouts bool, compare func(a, b slot) int) *inMemorySlotIndex {
	q := &inMemorySlotIndex{
		byTaskID: make(map[int64]slotLocation),
	}

	prioritySetIndex := func(s slot, i int) {
		q.setPriorityIndex(s.taskId, i)
	}

	priorityGetIndex := func(s slot) (int, bool) {
		return q.priorityIndex(s.taskId)
	}

	q.priorityQueue = newHeap(compare, prioritySetIndex, priorityGetIndex)

	if trackTimeouts {
		// ordering: earliest timeout first, then lower taskId.
		timedOutCompare := func(a, b timeoutEntry) int {
			if a.timeoutAtMs != b.timeoutAtMs {
				return cmp.Compare(a.timeoutAtMs, b.timeoutAtMs)
			}
			return cmp.Compare(a.taskId, b.taskId)
		}

		timedOutSetIndex := func(e timeoutEntry, i int) {
			q.setTimeoutIndex(e.taskId, i)
		}

		timedOutGetIndex := func(e timeoutEntry) (int, bool) {
			return q.timeoutIndex(e.taskId)
		}

		q.timedOutQueue = newHeap(timedOutCompare, timedOutSetIndex, timedOutGetIndex)
	}

	return q
}

func (q *inMemorySlotIndex) insert(s slot) {
	// drop any existing entry for this task so re-inserts (e.g. retries) don't duplicate.
	q.removeFromPriority(s.taskId)
	q.removeFromTimedOut(s.taskId)

	q.priorityQueue.insert(s)
	if q.timedOutQueue != nil {
		q.timedOutQueue.insert(timeoutEntry{taskId: s.taskId, timeoutAtMs: s.scheduleTimeoutAtMs})
	}
}

func (q *inMemorySlotIndex) delete(taskId int64) (slot, bool) {
	i, ok := q.priorityIndex(taskId)
	if !ok {
		return slot{}, false
	}
	s := q.priorityQueue.values[i]
	q.priorityQueue.delete(i)
	q.removeFromTimedOut(taskId)
	return s, true
}

func (q *inMemorySlotIndex) get(taskId int64) (slot, bool) {
	i, ok := q.priorityIndex(taskId)
	if !ok {
		return slot{}, false
	}
	return q.priorityQueue.values[i], true
}

func (q *inMemorySlotIndex) popTimedOut(now time.Time) []slot {
	if q.timedOutQueue == nil {
		return nil
	}

	nowMs := now.UnixMilli()

	var timedOut []slot
	for {
		e, ok := q.timedOutQueue.peek()
		if !ok || e.timeoutAtMs > nowMs {
			break
		}
		q.timedOutQueue.takeMin()
		// the full slot lives in the priority queue; fetch it before removing so callers get the
		// complete slot (taskInsertedAt, retryCount) they need to cancel it downstream.
		if i, ok := q.priorityIndex(e.taskId); ok {
			timedOut = append(timedOut, q.priorityQueue.values[i])
			q.priorityQueue.delete(i)
		}
	}
	return timedOut
}

func (q *inMemorySlotIndex) pop(n int) []slot {
	popped := q.priorityQueue.pop(n)
	for _, s := range popped {
		q.removeFromTimedOut(s.taskId)
	}
	return popped
}

func (q *inMemorySlotIndex) peek() (slot, bool) {
	return q.priorityQueue.peek()
}

func (q *inMemorySlotIndex) len() int {
	return q.priorityQueue.len()
}

func (q *inMemorySlotIndex) begin() {
	q.priorityQueue.begin()
	if q.timedOutQueue != nil {
		q.timedOutQueue.begin()
	}
}

func (q *inMemorySlotIndex) commit() {
	q.priorityQueue.commit()
	if q.timedOutQueue != nil {
		q.timedOutQueue.commit()
	}
}

func (q *inMemorySlotIndex) rollback() {
	q.priorityQueue.rollback()
	if q.timedOutQueue != nil {
		q.timedOutQueue.rollback()
	}
}

// removeFromPriority removes the task from the priority queue if present.
func (q *inMemorySlotIndex) removeFromPriority(taskId int64) {
	if i, ok := q.priorityIndex(taskId); ok {
		q.priorityQueue.delete(i)
	}
}

// removeFromTimedOut removes the task from the timeout queue if present.
func (q *inMemorySlotIndex) removeFromTimedOut(taskId int64) {
	if q.timedOutQueue == nil {
		return
	}
	if i, ok := q.timeoutIndex(taskId); ok {
		q.timedOutQueue.delete(i)
	}
}

// priorityIndex returns the task's index in the priority queue, or ok=false if absent.
func (q *inMemorySlotIndex) priorityIndex(taskId int64) (int, bool) {
	loc, ok := q.byTaskID[taskId]
	if !ok || loc.priIdx == 0 {
		return 0, false
	}
	return loc.priIdx - 1, true
}

// timeoutIndex returns the task's index in the timeout queue, or ok=false if absent.
func (q *inMemorySlotIndex) timeoutIndex(taskId int64) (int, bool) {
	loc, ok := q.byTaskID[taskId]
	if !ok || loc.toIdx == 0 {
		return 0, false
	}
	return loc.toIdx - 1, true
}

// setPriorityIndex records (i>=0) or clears (i<0) the task's priority-queue index, dropping the map
// entry once the task is absent from both heaps.
func (q *inMemorySlotIndex) setPriorityIndex(taskId int64, i int) {
	loc := q.byTaskID[taskId]
	loc.priIdx = i + 1 // i<0 -> 0 (absent)
	if loc.priIdx == 0 && loc.toIdx == 0 {
		delete(q.byTaskID, taskId)
		return
	}
	q.byTaskID[taskId] = loc
}

// setTimeoutIndex records (i>=0) or clears (i<0) the task's timeout-queue index, dropping the map
// entry once the task is absent from both heaps.
func (q *inMemorySlotIndex) setTimeoutIndex(taskId int64, i int) {
	loc := q.byTaskID[taskId]
	loc.toIdx = i + 1 // i<0 -> 0 (absent)
	if loc.priIdx == 0 && loc.toIdx == 0 {
		delete(q.byTaskID, taskId)
		return
	}
	q.byTaskID[taskId] = loc
}
