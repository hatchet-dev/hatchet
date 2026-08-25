package concurrency

import (
	"context"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func newCancelExceptOldestStrategy(repo repository.ConcurrencyRepository, maxConcurrency int32) *ConcurrencyStrategy {
	return newTestStrategyKind(repo, maxConcurrency, sqlcv1.V1ConcurrencyStrategyCANCELEXCEPTOLDEST)
}

// Within a single batch: free capacity is filled from the queued backlog oldest-first, then
// everything queued beyond the oldest maxRuns survivors is cancelled.
func TestCancelExceptOldest_SingleBatchKeepsOnlyOldestQueued(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelExceptOldestStrategy(repo, 1)

	msgs := []walMessage{
		walInsert("a", 1, 5, now, future),                    // oldest overall: fills the only slot
		walInsert("a", 2, 5, now.Add(time.Second), future),   // oldest of what's left: survives queued
		walInsert("a", 3, 5, now.Add(2*time.Second), future), // superseded
	}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	filled := filledIDs(repo.lastFilled)
	if len(filled) != 1 || !containsID(filled, 1) {
		t.Fatalf("filled = %v, want [1]", filled)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 1 || !containsID(cancelled, 3) {
		t.Fatalf("cancelled = %v, want [3]", cancelled)
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.queued.len() != 1 {
		t.Fatalf("queued len = %d, want 1", sq.queued.len())
	}
	if _, ok := sq.queued.get(2); !ok {
		t.Fatalf("oldest queued survivor (2) not retained")
	}
}

// The core property under test: across separate decide() calls - one per arrival - the first queued
// arrival must persist indefinitely as the sole survivor, and every later arrival must be cancelled
// by the very call that supersedes it.
func TestCancelExceptOldest_AcrossSuccessiveDecideCalls(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelExceptOldestStrategy(repo, 1)
	sq := c.getOrCreateSubQueue("a")

	// task 0 arrives alone and occupies the only slot.
	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsert("a", 0, 5, now, future),
	}); err != nil {
		t.Fatalf("processWALMessages(insert 0): %v", err)
	}
	c.pruneEmpty(c.commitScopes())

	if _, ok := sq.running.get(0); !ok {
		t.Fatalf("task 0 should be running after the first call")
	}

	// task 1 becomes the sole queued survivor (the first to queue up).
	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsert("a", 1, 5, now.Add(time.Second), future),
	}); err != nil {
		t.Fatalf("processWALMessages(insert 1): %v", err)
	}
	c.pruneEmpty(c.commitScopes())

	if sq.queued.len() != 1 {
		t.Fatalf("queued len = %d, want 1", sq.queued.len())
	}
	if _, ok := sq.queued.get(1); !ok {
		t.Fatalf("task 1 should be the sole queued survivor")
	}

	// tasks 2..4 arrive one at a time; task 1 (the oldest queued) must survive every single call,
	// and each newcomer must be cancelled by the same call that superseded it - not just eventually.
	for i, id := range []int64{2, 3, 4} {
		if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
			walInsert("a", id, 5, now.Add(time.Duration(i+2)*time.Second), future),
		}); err != nil {
			t.Fatalf("processWALMessages(insert %d): %v", id, err)
		}

		if got := filledIDs(repo.lastFilled); len(got) != 0 {
			t.Fatalf("after inserting %d: filled = %v, want none (slot is occupied)", id, got)
		}
		cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
		if len(cancelled) != 1 || !containsID(cancelled, id) {
			t.Fatalf("after inserting %d: cancelled = %v, want [%d]", id, cancelled, id)
		}
		c.pruneEmpty(c.commitScopes())

		if sq.queued.len() != 1 {
			t.Fatalf("after inserting %d: queued len = %d, want 1", id, sq.queued.len())
		}
		if _, ok := sq.queued.get(1); !ok {
			t.Fatalf("after inserting %d: task 1 (oldest) must still be the sole queued survivor", id)
		}
		if sq.running.len() != 1 {
			t.Fatalf("after inserting %d: running len = %d, want 1 (task 0 must still be running)", id, sq.running.len())
		}
	}
}

// Once the running slot frees up, the surviving (oldest) queued run must be promoted by the very
// next decide() call.
func TestCancelExceptOldest_PromotesSurvivorAsSoonAsRunningSlotFrees(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelExceptOldestStrategy(repo, 1)

	msgs := []walMessage{
		walInsert("a", 0, 5, now, future),
		walInsert("a", 1, 5, now.Add(time.Second), future), // oldest queued, sole survivor
		walInsert("a", 2, 5, now.Add(2*time.Second), future),
	}
	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.pruneEmpty(c.commitScopes())

	sq := c.getOrCreateSubQueue("a")
	if _, ok := sq.running.get(0); !ok {
		t.Fatalf("task 0 should be running")
	}
	if _, ok := sq.queued.get(1); !ok {
		t.Fatalf("task 1 (oldest queued) should be the sole survivor")
	}

	// the occupying task completes and is removed from the index.
	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		{Operation: "DELETE", Key: "a", TaskId: 0},
	}); err != nil {
		t.Fatalf("processWALMessages (delete): %v", err)
	}

	filled := filledIDs(repo.lastFilled)
	if len(filled) != 1 || !containsID(filled, 1) {
		t.Fatalf("filled = %v, want [1] (the survivor must run as soon as the slot frees)", filled)
	}
	if _, ok := sq.running.get(1); !ok {
		t.Fatalf("task 1 should now be running")
	}
	if sq.queued.len() != 0 {
		t.Fatalf("queued len = %d, want 0", sq.queued.len())
	}
}

// At capacity, CANCEL_EXCEPT_OLDEST must never preempt running work.
func TestCancelExceptOldest_NeverPreemptsRunning(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 3, 0, now, future, true), // running
			indexRow("a", 2, 3, 0, now, future, true), // running
		},
	}
	c := newCancelExceptOldestStrategy(repo, 2)

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsert("a", 3, 9, now, future),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	if got := filledIDs(repo.lastFilled); len(got) != 0 {
		t.Fatalf("filled = %v, want none (running must not be preempted)", got)
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 2 {
		t.Fatalf("running len = %d, want 2", sq.running.len())
	}
	if _, ok := sq.running.get(1); !ok {
		t.Fatalf("running task 1 must be protected")
	}
	if _, ok := sq.running.get(2); !ok {
		t.Fatalf("running task 2 must be protected")
	}

	if sq.queued.len() != 1 {
		t.Fatalf("queued len = %d, want 1", sq.queued.len())
	}
	if _, ok := sq.queued.get(3); !ok {
		t.Fatalf("newcomer 3 should remain queued (nothing older is queued to prefer over it)")
	}
}

// maxRuns > 1 generalizes to keeping the oldest N queued survivors, not just one.
func TestCancelExceptOldest_KeepsOldestNWhenMaxRunsGreaterThanOne(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelExceptOldestStrategy(repo, 2)

	// 5 arrivals against 2 running slots: the two oldest fill running, leaving 3 queued - one more
	// than maxRuns, so exactly one (the newest of what's left) must be cancelled.
	msgs := []walMessage{
		walInsert("a", 0, 5, now, future),
		walInsert("a", 1, 5, now.Add(time.Second), future),
		walInsert("a", 2, 5, now.Add(2*time.Second), future),
		walInsert("a", 3, 5, now.Add(3*time.Second), future),
		walInsert("a", 4, 5, now.Add(4*time.Second), future),
	}
	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	// two oldest (0, 1) fill both running slots; among the remaining queued (2, 3, 4), the newest
	// of what's left (4) is cancelled and the oldest two (2, 3) survive.
	filled := filledIDs(repo.lastFilled)
	if len(filled) != 2 || !containsID(filled, 0) || !containsID(filled, 1) {
		t.Fatalf("filled = %v, want {0,1}", filled)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 1 || !containsID(cancelled, 4) {
		t.Fatalf("cancelled = %v, want [4]", cancelled)
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.queued.len() != 2 {
		t.Fatalf("queued len = %d, want 2", sq.queued.len())
	}
	if _, ok := sq.queued.get(2); !ok {
		t.Fatalf("oldest queued survivor (2) not retained")
	}
	if _, ok := sq.queued.get(3); !ok {
		t.Fatalf("oldest queued survivor (3) not retained")
	}
}
