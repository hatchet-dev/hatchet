package concurrency

import (
	"testing"
	"time"
)

var baseTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func mkSlot(taskId int64, priority int32, insertedAt, timeoutAt time.Time) slot {
	return slot{
		priority:            priority,
		taskId:              taskId,
		taskInsertedAtNs:    insertedAt.UnixNano(),
		scheduleTimeoutAtMs: timeoutAt.UnixMilli(),
	}
}

// checkConsistent asserts the cross-queue invariant: a task is present in the
// priority queue iff it is present in the timeout queue, and every index map
// entry points at the slot it claims to.
func checkConsistent(t *testing.T, q *inMemorySlotIndex) {
	t.Helper()

	if len(q.byTaskID) != q.priorityQueue.len() {
		t.Fatalf("index map size %d != priority heap size %d", len(q.byTaskID), q.priorityQueue.len())
	}

	tracking := q.timedOutQueue != nil
	if tracking && q.timedOutQueue.len() != q.priorityQueue.len() {
		t.Fatalf("timeout heap size %d != priority heap size %d", q.timedOutQueue.len(), q.priorityQueue.len())
	}

	for id := range q.byTaskID {
		i, ok := q.priorityIndex(id)
		if !ok {
			t.Fatalf("taskId %d has no priority-queue index", id)
		}
		if q.priorityQueue.values[i].taskId != id {
			t.Fatalf("priority index: key %d -> index %d holds taskId %d", id, i, q.priorityQueue.values[i].taskId)
		}

		if !tracking {
			continue
		}
		j, ok := q.timeoutIndex(id)
		if !ok {
			t.Fatalf("taskId %d in priority queue but missing timeout-queue index", id)
		}
		if q.timedOutQueue.values[j].taskId != id {
			t.Fatalf("timeout index: key %d -> index %d holds taskId %d", id, j, q.timedOutQueue.values[j].taskId)
		}
	}
}

func taskIDs(slots []slot) []int64 {
	ids := make([]int64, len(slots))
	for i, s := range slots {
		ids[i] = s.taskId
	}
	return ids
}

func equalIDs(a []int64, b ...int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestInsertGetLen(t *testing.T) {
	q := newInMemorySlotIndex(true)
	if q.len() != 0 {
		t.Fatalf("empty len = %d", q.len())
	}
	if _, ok := q.get(1); ok {
		t.Fatalf("get on empty returned ok")
	}

	q.insert(mkSlot(1, 5, baseTime, baseTime))
	q.insert(mkSlot(2, 5, baseTime, baseTime))
	checkConsistent(t, q)

	if q.len() != 2 {
		t.Fatalf("len = %d, want 2", q.len())
	}
	got, ok := q.get(1)
	if !ok || got.taskId != 1 {
		t.Fatalf("get(1) = %+v, %v", got, ok)
	}
}

func TestPopPriorityOrder(t *testing.T) {
	q := newInMemorySlotIndex(true)
	// Insert out of order; higher priority must pop first.
	q.insert(mkSlot(1, 1, baseTime, baseTime))
	q.insert(mkSlot(2, 9, baseTime, baseTime))
	q.insert(mkSlot(3, 5, baseTime, baseTime))

	got := q.pop(3)
	if !equalIDs(taskIDs(got), 2, 3, 1) {
		t.Fatalf("pop order by priority = %v, want [2 3 1]", taskIDs(got))
	}
	checkConsistent(t, q)
	if q.len() != 0 {
		t.Fatalf("len after draining = %d", q.len())
	}
}

func TestPopTieBreak(t *testing.T) {
	q := newInMemorySlotIndex(true)
	later := baseTime.Add(time.Minute)
	// Same priority: earlier taskInsertedAt wins, then lower taskId.
	q.insert(mkSlot(10, 5, later, baseTime))
	q.insert(mkSlot(20, 5, baseTime, baseTime)) // earliest inserted -> first
	q.insert(mkSlot(5, 5, later, baseTime))     // tie with taskId 10 on time -> lower id first

	got := q.pop(3)
	if !equalIDs(taskIDs(got), 20, 5, 10) {
		t.Fatalf("tie-break order = %v, want [20 5 10]", taskIDs(got))
	}
}

func TestPopPartial(t *testing.T) {
	q := newInMemorySlotIndex(true)
	for i := int64(1); i <= 5; i++ {
		q.insert(mkSlot(i, int32(i), baseTime, baseTime))
	}
	got := q.pop(2)
	if !equalIDs(taskIDs(got), 5, 4) {
		t.Fatalf("pop(2) = %v, want [5 4]", taskIDs(got))
	}
	checkConsistent(t, q)
	if q.len() != 3 {
		t.Fatalf("len = %d, want 3", q.len())
	}
}

func TestPopNonPositiveReturnsNil(t *testing.T) {
	q := newInMemorySlotIndex(true)
	q.insert(mkSlot(1, 1, baseTime, baseTime))
	if got := q.pop(0); got != nil {
		t.Fatalf("pop(0) = %v, want nil", got)
	}
	if got := q.pop(-3); got != nil {
		t.Fatalf("pop(-3) = %v, want nil", got)
	}
	if q.len() != 1 {
		t.Fatalf("len = %d, want 1 (nothing popped)", q.len())
	}
}

func TestInsertDedupesByTaskId(t *testing.T) {
	q := newInMemorySlotIndex(true)
	q.insert(mkSlot(1, 1, baseTime, baseTime))
	q.insert(mkSlot(1, 9, baseTime, baseTime)) // same taskId, updated priority

	checkConsistent(t, q)
	if q.len() != 1 {
		t.Fatalf("len = %d, want 1 after re-insert", q.len())
	}
	got, _ := q.get(1)
	if got.priority != 9 {
		t.Fatalf("get(1).priority = %d, want 9 (latest)", got.priority)
	}
}

func TestDeleteRemovesFromBothQueues(t *testing.T) {
	q := newInMemorySlotIndex(true)
	q.insert(mkSlot(1, 1, baseTime, baseTime))
	q.insert(mkSlot(2, 2, baseTime, baseTime))

	s, ok := q.delete(1)
	if !ok || s.taskId != 1 {
		t.Fatalf("delete(1) = %+v, %v", s, ok)
	}
	checkConsistent(t, q)
	if _, ok := q.get(1); ok {
		t.Fatalf("get(1) ok after delete")
	}
	if q.len() != 1 {
		t.Fatalf("len = %d, want 1", q.len())
	}
}

func TestDeleteMissingReturnsFalse(t *testing.T) {
	q := newInMemorySlotIndex(true)
	q.insert(mkSlot(1, 1, baseTime, baseTime))
	if _, ok := q.delete(99); ok {
		t.Fatalf("delete(99) returned ok for absent task")
	}
	checkConsistent(t, q)
}

func TestPopRemovesFromTimeoutQueue(t *testing.T) {
	q := newInMemorySlotIndex(true)
	// timeout far in the future so popTimedOut wouldn't otherwise drop them.
	future := baseTime.Add(time.Hour)
	q.insert(mkSlot(1, 5, baseTime, future))
	q.insert(mkSlot(2, 1, baseTime, future))

	q.pop(1) // removes the higher-priority task (id 1)
	checkConsistent(t, q)

	// The popped task must be gone from the timeout queue too.
	timedOut := q.popTimedOut(future.Add(time.Minute))
	if !equalIDs(taskIDs(timedOut), 2) {
		t.Fatalf("popTimedOut after pop = %v, want only [2]", taskIDs(timedOut))
	}
}

func TestPopTimedOutOrderAndRemoval(t *testing.T) {
	q := newInMemorySlotIndex(true)
	t1 := baseTime.Add(1 * time.Minute)
	t2 := baseTime.Add(2 * time.Minute)
	t3 := baseTime.Add(3 * time.Minute)
	q.insert(mkSlot(1, 5, baseTime, t3))
	q.insert(mkSlot(2, 5, baseTime, t1))
	q.insert(mkSlot(3, 5, baseTime, t2))

	// now is between t2 and t3: tasks 2 and 1 expire, in timeout order.
	got := q.popTimedOut(t2.Add(30 * time.Second))
	if !equalIDs(taskIDs(got), 2, 3) {
		t.Fatalf("popTimedOut = %v, want [2 3] (by timeout)", taskIDs(got))
	}
	checkConsistent(t, q)

	// Expired tasks are removed from the priority queue too.
	if _, ok := q.get(2); ok {
		t.Fatalf("expired task 2 still gettable")
	}
	if q.len() != 1 {
		t.Fatalf("len = %d, want 1 (only task 1 remains)", q.len())
	}
	if got, _ := q.get(1); got.taskId != 1 {
		t.Fatalf("remaining task = %+v, want taskId 1", got)
	}
}

func TestPopTimedOutBoundary(t *testing.T) {
	q := newInMemorySlotIndex(true)
	timeoutAt := baseTime.Add(time.Minute)
	q.insert(mkSlot(1, 5, baseTime, timeoutAt))

	// Strictly before the timeout: nothing expires.
	if got := q.popTimedOut(timeoutAt.Add(-time.Nanosecond)); len(got) != 0 {
		t.Fatalf("popTimedOut before timeout = %v, want empty", taskIDs(got))
	}
	// Exactly at the timeout: expires (gate is After(now)).
	if got := q.popTimedOut(timeoutAt); !equalIDs(taskIDs(got), 1) {
		t.Fatalf("popTimedOut at timeout = %v, want [1]", taskIDs(got))
	}
}

func TestPopTimedOutWhenNotTracking(t *testing.T) {
	q := newInMemorySlotIndex(false)
	q.insert(mkSlot(1, 5, baseTime, baseTime))
	if got := q.popTimedOut(baseTime.Add(time.Hour)); got != nil {
		t.Fatalf("popTimedOut without tracking = %v, want nil", got)
	}
	// Untracked index still works as a plain priority queue.
	if q.len() != 1 {
		t.Fatalf("len = %d, want 1", q.len())
	}
	checkConsistent(t, q)
}
