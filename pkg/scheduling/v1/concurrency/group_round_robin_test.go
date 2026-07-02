package concurrency

import (
	"context"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func newGroupRoundRobinStrategy(repo repository.ConcurrencyRepository, maxConcurrency int32) *ConcurrencyStrategy {
	return newTestStrategyKind(repo, maxConcurrency, sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN)
}

// Free capacity is filled from the queued backlog in comparator order (highest priority first); the
// slots that don't fit stay queued. GROUP_ROUND_ROBIN never cancels them (unlike CANCEL_NEWEST) and
// never preempts running work (unlike CANCEL_IN_PROGRESS).
func TestGroupRoundRobin_FillsFreeCapacityByPriority(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newGroupRoundRobinStrategy(repo, 2) // only 2 slots may run

	msgs := []walMessage{
		walInsert("a", 101, 1, now, future),
		walInsert("a", 105, 5, now, future),
		walInsert("a", 109, 9, now, future),
	}

	res, err := c.processWALMessages(context.Background(), nil, msgs)
	if err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	if res == nil {
		t.Fatalf("nil result")
	}

	// Highest priority fills first, up to MaxConcurrency; the rest stays queued.
	if got := filledIDs(repo.lastFilled); len(got) != 2 || got[0] != 109 || got[1] != 105 {
		t.Fatalf("filled = %v, want [109 105]", got)
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 2 {
		t.Fatalf("running len = %d, want 2", sq.running.len())
	}
	if sq.queued.len() != 1 {
		t.Fatalf("queued len = %d, want 1", sq.queued.len())
	}
	if _, ok := sq.queued.get(101); !ok {
		t.Fatalf("lowest-priority task 101 should remain queued")
	}

	// nothing is cancelled: the backlog that didn't fit stays queued.
	if got := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit); len(got) != 0 {
		t.Fatalf("unexpected CONCURRENCY_LIMIT cancellations: %v", got)
	}

	// mirror Run's success path
	c.commitScopes()
	if c.openScopes != nil {
		t.Fatalf("openScopes not cleared after commit")
	}
}

// Queued slots past their scheduling timeout are cancelled as SCHEDULING_TIMED_OUT and excluded from
// filling, so a timed-out slot is never promoted to running even if it outranks the survivors.
func TestGroupRoundRobin_TimedOutExcludedFromFill(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newGroupRoundRobinStrategy(repo, 5) // plenty of room; timeout, not capacity, is under test

	msgs := []walMessage{
		walInsert("a", 201, 9, now, past),   // already past its scheduling timeout
		walInsert("a", 202, 1, now, future), // still valid
	}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	// 201 timed out: cancelled with SCHEDULING_TIMED_OUT, never filled.
	timedOut := cancelledByReason(repo.lastCancelled, repository.CancelledReasonSchedulingTimedOut)
	if !containsID(timedOut, 201) {
		t.Fatalf("task 201 not cancelled as timed out: %v", timedOut)
	}
	filled := filledIDs(repo.lastFilled)
	if containsID(filled, 201) {
		t.Fatalf("timed-out task 201 was filled: %v", filled)
	}
	if !containsID(filled, 202) {
		t.Fatalf("valid task 202 was not filled: %v", filled)
	}

	sq := c.getOrCreateSubQueue("a")
	if _, ok := sq.queued.get(201); ok {
		t.Fatalf("timed-out task 201 still in queued index")
	}
	if _, ok := sq.running.get(202); !ok {
		t.Fatalf("task 202 not promoted to running")
	}
}
