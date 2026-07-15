package concurrency

// subQueue represents the queue for a specific concurrency key
type subQueue struct {
	running slotIndex
	queued  slotIndex
	compare func(a, b slot) int
	key     string
	maxRuns int32
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

// begin opens an undo scope across both indexes so the mutations made while processing a batch can be
// rolled back as a unit if the accompanying database flush fails.
func (s *subQueue) begin() {
	s.running.begin()
	s.queued.begin()
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
}
