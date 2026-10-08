package concurrency

import (
	"cmp"
	"slices"
)

// maxSmallRunning is the most running slots a sub-queue keeps in a sorted slice.
const maxSmallRunning = 64

// runningSlots holds a sub-queue's running slots. Most keys run a few slots, which are kept in a
// slice sorted by taskId: far smaller than a heap and a map. Once a key runs more than
// maxSmallRunning slots they move to a heap index ordered worst-first, so updates stay O(log n).
type runningSlots struct {
	small []slot
	large *inMemorySlotIndex
}

func (r *runningSlots) find(taskId int64) (int, bool) {
	return slices.BinarySearchFunc(r.small, taskId, func(s slot, id int64) int {
		return cmp.Compare(s.taskId, id)
	})
}

// insert adds s, replacing any slot with the same taskId. compare is the strategy's slot ordering.
func (r *runningSlots) insert(s slot, compare func(a, b slot) int) {
	if r.large != nil {
		r.large.insert(s)
		return
	}

	i, found := r.find(s.taskId)

	switch {
	case found:
		r.small[i] = s
	case len(r.small) < maxSmallRunning:
		r.small = slices.Insert(r.small, i, s)
	default:
		r.large = newInMemorySlotIndexWithCompare(false, reverseCompare(compare))

		for _, existing := range r.small {
			r.large.insert(existing)
		}

		r.large.insert(s)
		r.small = nil
	}
}

func (r *runningSlots) delete(taskId int64) (slot, bool) {
	if r.large != nil {
		return r.large.delete(taskId)
	}

	i, found := r.find(taskId)
	if !found {
		return slot{}, false
	}

	s := r.small[i]
	r.small = slices.Delete(r.small, i, i+1)

	return s, true
}

func (r *runningSlots) get(taskId int64) (slot, bool) {
	if r.large != nil {
		return r.large.get(taskId)
	}

	i, found := r.find(taskId)
	if !found {
		return slot{}, false
	}

	return r.small[i], true
}

func (r *runningSlots) len() int {
	if r.large != nil {
		return r.large.len()
	}

	return len(r.small)
}

// worst returns the slot ranked last under compare; ok is false if there are no running slots.
func (r *runningSlots) worst(compare func(a, b slot) int) (s slot, ok bool) {
	if r.large != nil {
		return r.large.peek()
	}

	if len(r.small) == 0 {
		return slot{}, false
	}

	return slices.MaxFunc(r.small, compare), true
}
