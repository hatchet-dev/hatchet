package concurrency

import (
	"context"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func newCancelExceptNewestStrategy(repo repository.ConcurrencyRepository, maxConcurrency int32) *ConcurrencyStrategy {
	return newTestStrategyKind(repo, maxConcurrency, sqlcv1.V1ConcurrencyStrategyCANCELEXCEPTNEWEST)
}

// Within a single batch: free capacity is filled from the queued backlog oldest-first, then
// everything queued beyond the newest maxRuns survivors is cancelled.
func TestCancelExceptNewest_SingleBatchKeepsOnlyNewestQueued(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelExceptNewestStrategy(repo, 1)

	msgs := []walMessage{
		walInsert("a", 1, 5, now, future),                    // oldest: fills the only slot
		walInsert("a", 2, 5, now.Add(time.Second), future),   // superseded
		walInsert("a", 3, 5, now.Add(2*time.Second), future), // newest: survives queued
	}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	filled := filledIDs(repo.lastFilled)
	if len(filled) != 1 || !containsID(filled, 1) {
		t.Fatalf("filled = %v, want [1]", filled)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 1 || !containsID(cancelled, 2) {
		t.Fatalf("cancelled = %v, want [2]", cancelled)
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.queued.len() != 1 {
		t.Fatalf("queued len = %d, want 1", sq.queued.len())
	}
	if _, ok := sq.queued.get(3); !ok {
		t.Fatalf("newest queued survivor (3) not retained")
	}
}

// The core property under test: across separate decide() calls - one per arrival, exactly as WAL
// messages land in production - only the single newest queued arrival must ever be alive at once.
// Each older arrival must be cancelled by the very call that supersedes it, not just eventually.
// This is the scenario that the `return toFill, nil` bug (discarding a correctly-computed toCancel)
// broke: superseded runs were dropped from the in-memory index without ever being cancelled in the
// DB, so they sat QUEUED forever.
func TestCancelExceptNewest_AcrossSuccessiveDecideCalls(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelExceptNewestStrategy(repo, 1)
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

	// tasks 1..3 arrive one at a time while 0 is still running. after each call, exactly the
	// newly-arrived task must be queued, its predecessor must be cancelled by that same call, and
	// task 0 must still be running (never preempted).
	prev := int64(-1)
	for i, id := range []int64{1, 2, 3} {
		if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
			walInsert("a", id, 5, now.Add(time.Duration(i+1)*time.Second), future),
		}); err != nil {
			t.Fatalf("processWALMessages(insert %d): %v", id, err)
		}

		if got := filledIDs(repo.lastFilled); len(got) != 0 {
			t.Fatalf("after inserting %d: filled = %v, want none (slot is occupied)", id, got)
		}
		if prev >= 0 {
			cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
			if len(cancelled) != 1 || !containsID(cancelled, prev) {
				t.Fatalf("after inserting %d: cancelled = %v, want [%d]", id, cancelled, prev)
			}
		}
		c.pruneEmpty(c.commitScopes())

		if sq.queued.len() != 1 {
			t.Fatalf("after inserting %d: queued len = %d, want 1", id, sq.queued.len())
		}
		if _, ok := sq.queued.get(id); !ok {
			t.Fatalf("after inserting %d: it should be the sole queued survivor", id)
		}
		if sq.running.len() != 1 {
			t.Fatalf("after inserting %d: running len = %d, want 1 (task 0 must still be running)", id, sq.running.len())
		}
		if _, ok := sq.running.get(0); !ok {
			t.Fatalf("after inserting %d: task 0 must still be the runner", id)
		}

		prev = id
	}
}

// Once the running slot frees up, the surviving queued run must be promoted by the very next
// decide() call - it must not need a further arrival on this key to be reconsidered.
func TestCancelExceptNewest_PromotesSurvivorAsSoonAsRunningSlotFrees(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelExceptNewestStrategy(repo, 1)

	msgs := []walMessage{
		walInsert("a", 0, 5, now, future),
		walInsert("a", 1, 5, now.Add(time.Second), future),
		walInsert("a", 2, 5, now.Add(2*time.Second), future), // newest, sole survivor
	}
	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.pruneEmpty(c.commitScopes())

	sq := c.getOrCreateSubQueue("a")
	if _, ok := sq.running.get(0); !ok {
		t.Fatalf("task 0 should be running")
	}
	if _, ok := sq.queued.get(2); !ok {
		t.Fatalf("task 2 (newest) should be the sole queued survivor")
	}

	// the occupying task completes and is removed from the index.
	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		{Operation: "DELETE", Key: "a", TaskId: 0},
	}); err != nil {
		t.Fatalf("processWALMessages (delete): %v", err)
	}

	filled := filledIDs(repo.lastFilled)
	if len(filled) != 1 || !containsID(filled, 2) {
		t.Fatalf("filled = %v, want [2] (the survivor must run as soon as the slot frees)", filled)
	}
	if _, ok := sq.running.get(2); !ok {
		t.Fatalf("task 2 should now be running")
	}
	if sq.queued.len() != 0 {
		t.Fatalf("queued len = %d, want 0", sq.queued.len())
	}
}

// At capacity, CANCEL_EXCEPT_NEWEST must never preempt running work, even across many arrivals -
// only the queued backlog is ever touched.
func TestCancelExceptNewest_NeverPreemptsRunning(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 3, 0, now, future, true), // running
			indexRow("a", 2, 3, 0, now, future, true), // running
		},
	}
	c := newCancelExceptNewestStrategy(repo, 2)

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	// a higher-priority newcomer arrives at a full group.
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

	// the newcomer is the sole queued survivor (nothing older is queued to be cancelled against).
	if sq.queued.len() != 1 {
		t.Fatalf("queued len = %d, want 1", sq.queued.len())
	}
	if _, ok := sq.queued.get(3); !ok {
		t.Fatalf("newcomer 3 should remain queued as the newest")
	}
}

// maxRuns > 1 generalizes to keeping the newest N queued survivors, not just one.
func TestCancelExceptNewest_KeepsNewestNWhenMaxRunsGreaterThanOne(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelExceptNewestStrategy(repo, 2)

	// 5 arrivals against 2 running slots: the two oldest fill running, leaving 3 queued - one more
	// than maxRuns, so exactly one (the oldest of what's left) must be cancelled.
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

	// two oldest (0, 1) fill both running slots; among the remaining queued (2, 3, 4), the oldest of
	// what's left (2) is cancelled and the newest two (3, 4) survive.
	filled := filledIDs(repo.lastFilled)
	if len(filled) != 2 || !containsID(filled, 0) || !containsID(filled, 1) {
		t.Fatalf("filled = %v, want {0,1}", filled)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 1 || !containsID(cancelled, 2) {
		t.Fatalf("cancelled = %v, want [2]", cancelled)
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.queued.len() != 2 {
		t.Fatalf("queued len = %d, want 2", sq.queued.len())
	}
	if _, ok := sq.queued.get(3); !ok {
		t.Fatalf("newest queued survivor (3) not retained")
	}
	if _, ok := sq.queued.get(4); !ok {
		t.Fatalf("newest queued survivor (4) not retained")
	}
}
