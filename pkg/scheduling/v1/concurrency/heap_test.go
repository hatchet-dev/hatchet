package concurrency

import (
	"slices"
	"testing"
)

// item is a test element with a stable identity (key) and an ordering value (pri).
type item struct {
	key int
	pri int
}

// newTestHeap returns a heap plus the external index map that setIndex maintains,
// mirroring how inMemorySlotIndex wires the heap to a taskId->index map.
func newTestHeap() (*heap[item], map[int]int) {
	idx := map[int]int{}
	setIndex := func(it item, i int) {
		if i < 0 {
			delete(idx, it.key)
			return
		}
		idx[it.key] = i
	}
	getIndex := func(it item) (int, bool) {
		i, ok := idx[it.key]
		return i, ok
	}
	compare := func(a, b item) int {
		if a.pri != b.pri {
			return cmpInt(a.pri, b.pri)
		}
		return cmpInt(a.key, b.key)
	}
	return newHeap(compare, setIndex, getIndex), idx
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// verify asserts the two structural invariants: the heap property holds, and the
// external index map exactly mirrors the values slice (no stale or missing keys).
func verify(t *testing.T, h *heap[item], idx map[int]int) {
	t.Helper()

	for i := range h.values {
		lc, rc := 2*i+1, 2*i+2
		if lc < len(h.values) && h.compare(h.values[i], h.values[lc]) > 0 {
			t.Fatalf("heap property violated at %d (left child): %+v > %+v", i, h.values[i], h.values[lc])
		}
		if rc < len(h.values) && h.compare(h.values[i], h.values[rc]) > 0 {
			t.Fatalf("heap property violated at %d (right child): %+v > %+v", i, h.values[i], h.values[rc])
		}
	}

	if len(idx) != len(h.values) {
		t.Fatalf("index map size %d != heap size %d (stale or missing entries): map=%v values=%v", len(idx), len(h.values), idx, h.values)
	}
	for i, v := range h.values {
		got, ok := idx[v.key]
		if !ok {
			t.Fatalf("key %d at index %d missing from index map", v.key, i)
		}
		if got != i {
			t.Fatalf("index map for key %d = %d, want %d", v.key, got, i)
		}
	}
}

func TestNewHeapPanicsOnNilFuncs(t *testing.T) {
	noopGetIndex := func(item) (int, bool) { return 0, false }
	assertPanics(t, "nil compare", func() {
		newHeap(nil, func(item, int) {}, noopGetIndex)
	})
	assertPanics(t, "nil setIndex", func() {
		newHeap(func(a, b item) int { return 0 }, nil, noopGetIndex)
	})
	assertPanics(t, "nil getIndex", func() {
		newHeap(func(a, b item) int { return 0 }, func(item, int) {}, nil)
	})
}

func TestInsertMaintainsInvariants(t *testing.T) {
	h, idx := newTestHeap()

	// Insert in descending priority so every insert triggers up-swaps.
	for p := 10; p >= 1; p-- {
		h.insert(item{key: p, pri: p})
		verify(t, h, idx)
	}

	if h.len() != 10 {
		t.Fatalf("len = %d, want 10", h.len())
	}
	if h.values[0].pri != 1 {
		t.Fatalf("min pri = %d, want 1", h.values[0].pri)
	}
}

func TestPopReturnsSortedOrder(t *testing.T) {
	h, idx := newTestHeap()
	for _, p := range []int{5, 3, 8, 1, 9, 2, 7} {
		h.insert(item{key: p, pri: p})
	}

	got := h.pop(h.len())
	var gotPri []int
	for _, it := range got {
		gotPri = append(gotPri, it.pri)
	}
	want := []int{1, 2, 3, 5, 7, 8, 9}
	if !slices.Equal(gotPri, want) {
		t.Fatalf("pop order = %v, want %v", gotPri, want)
	}
	verify(t, h, idx)
}

func TestPopPartialAndOverflow(t *testing.T) {
	h, idx := newTestHeap()
	for _, p := range []int{4, 2, 6, 1, 3} {
		h.insert(item{key: p, pri: p})
	}

	first := h.pop(2)
	if len(first) != 2 || first[0].pri != 1 || first[1].pri != 2 {
		t.Fatalf("pop(2) = %+v, want pris [1 2]", first)
	}
	verify(t, h, idx)

	// pop more than remaining returns only what's left.
	rest := h.pop(100)
	if len(rest) != 3 {
		t.Fatalf("pop(100) returned %d, want 3", len(rest))
	}
	verify(t, h, idx)

	if got := h.pop(5); len(got) != 0 {
		t.Fatalf("pop on empty = %+v, want empty", got)
	}
}

func TestPopZero(t *testing.T) {
	h, idx := newTestHeap()
	h.insert(item{key: 1, pri: 1})
	if got := h.pop(0); len(got) != 0 {
		t.Fatalf("pop(0) = %+v, want empty", got)
	}
	verify(t, h, idx)
}

func TestTakeMinDrainsInOrder(t *testing.T) {
	h, idx := newTestHeap()
	for _, p := range []int{3, 1, 2} {
		h.insert(item{key: p, pri: p})
	}

	var got []int
	for h.len() > 0 {
		got = append(got, h.takeMin().pri)
		verify(t, h, idx)
	}
	if !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("takeMin order = %v, want [1 2 3]", got)
	}
}

func TestTakeMinPanicsOnEmpty(t *testing.T) {
	h, _ := newTestHeap()
	assertPanics(t, "takeMin empty", func() { h.takeMin() })
}

func TestPeek(t *testing.T) {
	h, idx := newTestHeap()

	if _, ok := h.peek(); ok {
		t.Fatalf("peek on empty heap returned ok")
	}

	for _, p := range []int{5, 3, 8, 1, 9} {
		h.insert(item{key: p, pri: p})
	}

	// peek returns the minimum without removing it.
	got, ok := h.peek()
	if !ok || got.pri != 1 {
		t.Fatalf("peek = %+v, %v, want min pri 1", got, ok)
	}
	if h.len() != 5 {
		t.Fatalf("peek mutated the heap: len = %d, want 5", h.len())
	}
	verify(t, h, idx)

	// after removing the min, peek reflects the new minimum.
	h.takeMin()
	got, ok = h.peek()
	if !ok || got.pri != 3 {
		t.Fatalf("peek after takeMin = %+v, %v, want min pri 3", got, ok)
	}
}

func TestDeleteMiddleDownPath(t *testing.T) {
	h, idx := newTestHeap()
	for _, p := range []int{1, 2, 3, 4, 5, 6, 7} {
		h.insert(item{key: p, pri: p})
	}

	// Delete a node near the root so the replacement sifts down.
	i := idx[2]
	h.delete(i)
	verify(t, h, idx)
	if _, ok := idx[2]; ok {
		t.Fatalf("key 2 still present in index map after delete")
	}
}

func TestDeleteCausesUpPath(t *testing.T) {
	h, idx := newTestHeap()
	// Construct a shape where the moved last element is smaller than the
	// deleted node's parent, forcing an up-sift.
	for _, p := range []int{1, 10, 2, 11, 12, 3, 4} {
		h.insert(item{key: p, pri: p})
	}

	i := idx[11]
	h.delete(i)
	verify(t, h, idx)
}

func TestDeleteLastElement(t *testing.T) {
	h, idx := newTestHeap()
	for _, p := range []int{1, 2, 3} {
		h.insert(item{key: p, pri: p})
	}

	last := len(h.values) - 1
	h.delete(last)
	verify(t, h, idx)
	if h.len() != 2 {
		t.Fatalf("len = %d, want 2", h.len())
	}
}

func TestDeleteIndexZero(t *testing.T) {
	h, idx := newTestHeap()
	for _, p := range []int{1, 2, 3, 4} {
		h.insert(item{key: p, pri: p})
	}
	h.delete(0)
	verify(t, h, idx)
	if _, ok := idx[1]; ok {
		t.Fatalf("min key 1 still in index map after delete(0)")
	}
}

func TestInsertManyIntoEmpty(t *testing.T) {
	h, idx := newTestHeap()
	h.insertMany(slices.Values([]item{
		{key: 5, pri: 5}, {key: 1, pri: 1}, {key: 3, pri: 3}, {key: 2, pri: 2}, {key: 4, pri: 4},
	}))
	verify(t, h, idx)

	got := h.pop(h.len())
	var gotPri []int
	for _, it := range got {
		gotPri = append(gotPri, it.pri)
	}
	if !slices.Equal(gotPri, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("insertMany then pop = %v, want sorted", gotPri)
	}
}

func TestInsertManyIntoNonEmpty(t *testing.T) {
	h, idx := newTestHeap()
	h.insert(item{key: 6, pri: 6})
	h.insert(item{key: 1, pri: 1})

	h.insertMany(slices.Values([]item{{key: 9, pri: 9}, {key: 2, pri: 2}, {key: 4, pri: 4}}))
	verify(t, h, idx)
	if h.len() != 5 {
		t.Fatalf("len = %d, want 5", h.len())
	}
}

func TestInsertManyEmptySeq(t *testing.T) {
	h, idx := newTestHeap()
	h.insert(item{key: 1, pri: 1})
	h.insertMany(slices.Values([]item{}))
	verify(t, h, idx)
}

// TestReinsertPoppedKey exercises the queue-level pattern: a key is popped, then
// re-inserted later (e.g. a retry). This is the path that the index-map bug breaks.
func TestReinsertPoppedKey(t *testing.T) {
	h, idx := newTestHeap()
	for _, p := range []int{1, 2, 3} {
		h.insert(item{key: p, pri: p})
	}

	min := h.takeMin()
	verify(t, h, idx)

	if _, ok := idx[min.key]; ok {
		t.Fatalf("popped key %d left a stale index entry: %v", min.key, idx)
	}
}

// membership returns the heap's keys sorted, so two heaps can be compared by the set of elements
// they hold regardless of internal array order.
func membership(h *heap[item]) []int {
	keys := make([]int, 0, len(h.values))
	for _, v := range h.values {
		keys = append(keys, v.key)
	}
	slices.Sort(keys)
	return keys
}

func TestNoRecordingOutsideScope(t *testing.T) {
	h, _ := newTestHeap()
	h.insert(item{key: 1, pri: 1})
	h.delete(0)
	if len(h.journal) != 0 {
		t.Fatalf("writes outside a scope recorded %d journal entries, want 0", len(h.journal))
	}
}

func TestCommitDiscardsLog(t *testing.T) {
	h, idx := newTestHeap()
	for _, p := range []int{3, 1, 2} {
		h.insert(item{key: p, pri: p})
	}

	h.begin()
	h.insert(item{key: 9, pri: 9})
	h.delete(idx[1])
	h.commit()

	if len(h.journal) != 0 || h.recording {
		t.Fatalf("commit left journal=%d recording=%v, want empty and not recording", len(h.journal), h.recording)
	}

	// rollback after commit must be a no-op: the committed state stands.
	before := membership(h)
	h.rollback()
	if !slices.Equal(membership(h), before) {
		t.Fatalf("rollback after commit changed membership: got %v, want %v", membership(h), before)
	}
	verify(t, h, idx)
}

func TestRollbackRestoresMixedOps(t *testing.T) {
	h, idx := newTestHeap()
	for _, p := range []int{5, 3, 8, 1, 9, 2, 7} {
		h.insert(item{key: p, pri: p})
	}
	want := membership(h)

	h.begin()
	h.insert(item{key: 100, pri: 100})
	h.insert(item{key: 0, pri: 0})
	h.delete(idx[8])
	h.pop(3)
	h.takeMin()
	h.rollback()

	if got := membership(h); !slices.Equal(got, want) {
		t.Fatalf("rollback membership = %v, want %v", got, want)
	}
	if h.recording || len(h.journal) != 0 {
		t.Fatalf("rollback left recording=%v journal=%d, want false and 0", h.recording, len(h.journal))
	}
	verify(t, h, idx)
}

// TestRollbackInsertThenDeleteSameKey: a key inserted and removed within the same scope must be
// absent after rollback, since it wasn't present at begin().
func TestRollbackInsertThenDeleteSameKey(t *testing.T) {
	h, idx := newTestHeap()
	h.insert(item{key: 1, pri: 1})
	want := membership(h)

	h.begin()
	h.insert(item{key: 42, pri: 42})
	h.delete(idx[42])
	h.rollback()

	if got := membership(h); !slices.Equal(got, want) {
		t.Fatalf("rollback membership = %v, want %v", got, want)
	}
	verify(t, h, idx)
}

// TestRollbackRetryReplace mirrors the strategy retry path: within one scope an existing key is
// deleted and re-inserted under the same identity (different pri). Rollback must restore the
// original element.
func TestRollbackRetryReplace(t *testing.T) {
	h, idx := newTestHeap()
	h.insert(item{key: 7, pri: 1})
	h.insert(item{key: 8, pri: 5})

	h.begin()
	h.delete(idx[7])
	h.insert(item{key: 7, pri: 99})
	h.rollback()

	// key 7 is back, and at its original priority (so it sits at the root again).
	i, ok := idx[7]
	if !ok {
		t.Fatalf("key 7 missing after rollback")
	}
	if h.values[i].pri != 1 {
		t.Fatalf("key 7 pri = %d after rollback, want 1 (original)", h.values[i].pri)
	}
	verify(t, h, idx)
}

func assertPanics(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s: expected panic, got none", name)
		}
	}()
	fn()
}
