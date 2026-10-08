package concurrency

// subQueue represents the queue for a specific concurrency key. A large backlog can hold millions
// of keys, so it stays small: most keys only have running slots.
type subQueue struct {
	key     string
	running runningSlots
	// queued is created on the first enqueue and dropped when a committed batch leaves it empty.
	queued  *inMemorySlotIndex
	compare func(a, b slot) int
	// undo is set between begin and commit or rollback.
	undo *subQueueUndo
	// maxRunsFrom is the task-inserted-at (ns) of the observation that set maxRuns, so
	// only a newer task's evaluation can change the limit.
	maxRunsFrom int64
	// maxRuns starts at the strategy's static max_concurrency and is overwritten by
	// observeMaxRuns when slots carry a dynamically evaluated value.
	maxRuns int32
}

// subQueueUndo is what rollback restores. The queued index records its own undo log.
type subQueueUndo struct {
	running     []undoEntry[slot]
	queued      *inMemorySlotIndex
	size        residentSize
	maxRunsFrom int64
	maxRuns     int32
}

func newSubQueue(key string, maxRuns int32, compare func(a, b slot) int) *subQueue {
	return &subQueue{
		key:     key,
		maxRuns: maxRuns,
		compare: compare,
	}
}

func (s *subQueue) slotsToRun() int32 {
	return s.maxRuns - int32(s.running.len()) //nolint:gosec // running slot count is bounded well within int32
}

// addRunning adds a slot to the running set, replacing any slot for the same task.
func (s *subQueue) addRunning(sl slot) {
	s.removeRunning(sl.taskId)
	s.running.insert(sl, s.compare)
	s.recordRunning(sl, true)
}

func (s *subQueue) removeRunning(taskId int64) (slot, bool) {
	sl, ok := s.running.delete(taskId)
	if ok {
		s.recordRunning(sl, false)
	}

	return sl, ok
}

func (s *subQueue) recordRunning(sl slot, added bool) {
	if s.undo != nil {
		s.undo.running = append(s.undo.running, undoEntry[slot]{value: sl, added: added})
	}
}

// enqueue adds a slot to the queued index, creating the index if the key had nothing queued.
func (s *subQueue) enqueue(sl slot) {
	if s.queued == nil {
		s.queued = newInMemorySlotIndexWithCompare(true, s.compare)
	}

	s.queued.insert(sl)
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

// begin opens an undo scope so the mutations made while processing a batch can be rolled back as a
// unit if the accompanying database flush fails.
func (s *subQueue) begin() {
	s.undo = &subQueueUndo{
		queued:      s.queued,
		size:        s.size(),
		maxRunsFrom: s.maxRunsFrom,
		maxRuns:     s.maxRuns,
	}

	if s.queued != nil {
		s.queued.begin()
	}
}

// commit discards the undo state once the database flush has succeeded, and returns how the
// sub-queue's size changed since begin.
func (s *subQueue) commit() residentSize {
	if s.undo == nil {
		return residentSize{}
	}

	before := s.undo.size
	s.undo = nil

	if s.queued != nil {
		s.queued.commit()

		if s.queued.len() == 0 {
			s.queued = nil
		}
	}

	return s.size().minus(before)
}

// rollback reverts every mutation made since begin, restoring the in-memory index to match the
// database after a failed flush. A queued index created during the scope is dropped, but running
// slots that moved to a heap index stay there, so it returns how the sub-queue's size changed.
func (s *subQueue) rollback() residentSize {
	u := s.undo
	if u == nil {
		return residentSize{}
	}

	s.undo = nil

	for i := len(u.running) - 1; i >= 0; i-- {
		if e := u.running[i]; e.added {
			s.running.delete(e.value.taskId)
		} else {
			s.running.insert(e.value, s.compare)
		}
	}

	s.queued = u.queued

	if s.queued != nil {
		s.queued.rollback()
	}

	s.maxRuns = u.maxRuns
	s.maxRunsFrom = u.maxRunsFrom

	return s.size().minus(u.size)
}

func (s *subQueue) size() residentSize {
	size := residentSize{
		running: int64(s.running.len()),
		queued:  int64(s.queued.len()),
	}

	if s.running.large != nil {
		size.largeRunning = size.running
	}

	if s.queued != nil {
		size.queuedKeys = 1
	}

	return size
}
