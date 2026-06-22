package concurrency

import (
	"iter"
	"slices"
)

// heap implements a typed min-heap with support for undo operations.
// inspired by `container/heap` and https://github.com/jba/heap, implemented with different signatures.
type heap[T any] struct {
	values  []T
	compare func(T, T) int

	// setIndex is used so callers can know the index of a value in the heap, for O(1) access
	setIndex func(T, int)
	// getIndex is the inverse of setIndex: it returns the current index of a value (matched by the
	// caller's identity, e.g. a taskId), or ok=false if the value is not in the heap. it is required
	// for rollback, which must locate an inserted value in order to remove it.
	getIndex func(T) (int, bool)

	// journal is the undo log. while recording, every membership change (insert/delete) appends the
	// inverse operation here so the caller can rollback() the whole scope. see begin/commit/rollback.
	journal   []undoEntry[T]
	recording bool
}

// undoEntry records a single membership change so it can be reversed. added=true means a value was
// inserted (undo: remove it); added=false means a value was removed (undo: re-insert it).
type undoEntry[T any] struct {
	value T
	added bool
}

func newHeap[T any](compare func(T, T) int, setIndex func(T, int), getIndex func(T) (int, bool)) *heap[T] {
	if compare == nil {
		panic("heap: compare function cannot be nil")
	}
	if setIndex == nil {
		panic("heap: setIndex function cannot be nil")
	}
	if getIndex == nil {
		panic("heap: getIndex function cannot be nil")
	}
	return &heap[T]{compare: compare, setIndex: setIndex, getIndex: getIndex}
}

// begin opens an undo scope. subsequent write operations append their inverse to the journal until
// the scope is closed with commit (discard the log) or rollback (apply it). outside a scope, writes
// record nothing, so paths that never undo (e.g. index builds) pay no overhead.
func (h *heap[T]) begin() {
	h.journal = h.journal[:0]
	h.recording = true
}

// commit closes the undo scope and discards the journal, making the recorded writes permanent from
// the heap's perspective.
func (h *heap[T]) commit() {
	h.journal = h.journal[:0]
	h.recording = false
}

// rollback closes the undo scope and reverses every recorded change in LIFO order, restoring the
// membership the heap had when begin() was called. it restores the heap invariant, not the exact
// internal array order (which does not affect correctness for a total-order compare).
func (h *heap[T]) rollback() {
	// stop recording first so the reversing writes don't themselves get journaled.
	h.recording = false
	for i := len(h.journal) - 1; i >= 0; i-- {
		e := h.journal[i]
		if e.added {
			if idx, ok := h.getIndex(e.value); ok {
				h.delete(idx)
			}
		} else {
			h.insert(e.value)
		}
	}
	h.journal = h.journal[:0]
}

func (h *heap[T]) insert(value T) {
	h.values = append(h.values, value)
	h.setIndex(value, len(h.values)-1)
	h.up(len(h.values) - 1)
	if h.recording {
		h.journal = append(h.journal, undoEntry[T]{added: true, value: value})
	}
}

// insertMany adds all elements of the sequence to the heap,
// re-establishing the heap property at the end.
func (h *heap[T]) insertMany(seq iter.Seq[T]) {
	start := len(h.values)
	h.values = slices.AppendSeq(h.values, seq)
	for i, e := range h.values[start:] {
		h.setIndex(e, start+i)
		if h.recording {
			h.journal = append(h.journal, undoEntry[T]{added: true, value: e})
		}
	}
	h.heapify()
}

// pop removes and returns the n smallest elements, in ascending order.
// Fewer are returned if the heap holds fewer than n.
func (h *heap[T]) pop(n int) []T {
	if n <= 0 {
		return nil
	}
	slots := make([]T, 0, min(n, h.len()))
	for i := 0; i < n && h.len() > 0; i++ {
		slots = append(slots, h.takeMin())
	}
	return slots
}

// peek returns the minimum element without removing it. ok is false if the heap is empty.
func (h *heap[T]) peek() (value T, ok bool) {
	if len(h.values) == 0 {
		return value, false
	}
	return h.values[0], true
}

func (h *heap[T]) takeMin() T {
	if len(h.values) == 0 {
		panic("heap: takeMin called on empty heap")
	}
	minVal := h.values[0]
	h.delete(0)
	return minVal
}

func (h *heap[T]) delete(i int) {
	if i < 0 || i >= len(h.values) {
		panic("heap: delete: index out of range")
	}

	removed := h.values[i]

	n := len(h.values) - 1
	if n != i {
		h.swap(i, n)
	}
	h.setIndex(h.values[n], -1) // the deleted element now sits at the tail
	var zero T
	h.values[n] = zero // allow GC
	h.values = h.values[:n]
	if n != i && !h.down(i) {
		h.up(i)
	}
	if h.recording {
		h.journal = append(h.journal, undoEntry[T]{added: false, value: removed})
	}
}

func (h *heap[T]) len() int {
	return len(h.values)
}

func (h *heap[T]) heapify() {
	for i := len(h.values)/2 - 1; i >= 0; i-- {
		h.down(i)
	}
}

// up moves the element at index i up the heap until the heap property
// is restored.
func (h *heap[T]) up(i int) {
	for i > 0 {
		p := (i - 1) / 2 // parent
		if h.compare(h.values[i], h.values[p]) >= 0 {
			break
		}
		h.swap(p, i)
		i = p
	}
}

// down moves the element at index i down the heap until the heap property
// is restored. It returns true if the element moved.
func (h *heap[T]) down(i int) bool {
	n := len(h.values)
	i0 := i
	for {
		lc := 2*i + 1
		if lc >= n {
			break
		}
		child := lc // left child
		if rc := lc + 1; rc < n && h.compare(h.values[rc], h.values[lc]) < 0 {
			child = rc // right child is smaller
		}
		if h.compare(h.values[child], h.values[i]) >= 0 {
			break
		}
		h.swap(i, child)
		i = child
	}
	return i > i0
}

func (h *heap[T]) swap(i, j int) {
	h.values[i], h.values[j] = h.values[j], h.values[i]
	h.setIndex(h.values[i], i)
	h.setIndex(h.values[j], j)
}
