package concurrency

// subQueue represents the queue for a specific concurrency key
type subQueue struct {
	running slotIndex
	queued  slotIndex
	compare func(a, b slot) int
	key     string
	// maxRuns starts at the strategy's static max_concurrency and is overwritten by
	// observeMaxRuns when slots carry a dynamically evaluated value.
	maxRuns int32
	// maxRunsFrom is the task-inserted-at (ns) of the observation that set maxRuns, so
	// only a newer task's evaluation can change the limit.
	maxRunsFrom int64

	// begin-scope snapshot: the membership undo journals on the slot indexes cannot cover
	// these scalars, so rollback restores them explicitly.
	maxRunsAtBegin     int32
	maxRunsFromAtBegin int64
}

func newSubQueue(key string, maxRuns int32, compare func(a, b slot) int) *subQueue {
	return &subQueue{
		key:     key,
		maxRuns: maxRuns,
		compare: compare,
		running: newInMemorySlotIndexWithCompare(false, reverseCompare(compare)),
		queued:  newInMemorySlotIndexWithCompare(true, compare),
	}
}

func (s *subQueue) slotsToRun() int32 {
	return s.maxRuns - int32(s.running.len()) //nolint:gosec // running slot count is bounded well within int32
}

// observeMaxRuns applies a slot's insert-time max-runs evaluation. The newest task's
// value wins: a re-inserted slot for an older task (replay, retry requeue) carries the
// original task timestamp and cannot regress a limit set by a newer task. Ties go to the
// later observation so same-instant inserts stay last-write-wins.
func (s *subQueue) observeMaxRuns(maxRuns int32, taskInsertedAtNs int64) {
	if taskInsertedAtNs < s.maxRunsFrom {
		return
	}

	s.maxRuns = maxRuns
	s.maxRunsFrom = taskInsertedAtNs
}

// begin opens an undo scope across both indexes so the mutations made while processing a batch can be
// rolled back as a unit if the accompanying database flush fails.
func (s *subQueue) begin() {
	s.running.begin()
	s.queued.begin()
	s.maxRunsAtBegin = s.maxRuns
	s.maxRunsFromAtBegin = s.maxRunsFrom
}

// commit discards the undo log once the database flush has succeeded.
func (s *subQueue) commit() {
	s.running.commit()
	s.queued.commit()
}

// rollback reverts every mutation recorded since begin, restoring the in-memory index to match the
// database after a failed flush.
func (s *subQueue) rollback() {
	s.running.rollback()
	s.queued.rollback()
	s.maxRuns = s.maxRunsAtBegin
	s.maxRunsFrom = s.maxRunsFromAtBegin
}
