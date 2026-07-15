package concurrency

import (
	"context"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func newCancelNewestStrategy(repo repository.ConcurrencyRepository, maxConcurrency int32) *ConcurrencyStrategy {
	return newTestStrategyKind(repo, maxConcurrency, sqlcv1.V1ConcurrencyStrategyCANCELNEWEST)
}

// Free capacity is filled from the queued backlog in comparator order; the slots that don't fit are
// cancelled with CONCURRENCY_LIMIT.
func TestCancelNewest_FillsFreeCapacityByPriority(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelNewestStrategy(repo, 2)

	msgs := []walMessage{
		walInsert("a", 10, 1, now, future), // lowest priority
		walInsert("a", 11, 5, now, future),
		walInsert("a", 12, 9, now, future), // highest priority
	}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	// two highest priority (12, 11) fill the two slots; lowest (10) is cancelled
	filled := filledIDs(repo.lastFilled)
	if len(filled) != 2 || !containsID(filled, 12) || !containsID(filled, 11) {
		t.Fatalf("filled = %v, want {11,12}", filled)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 1 || !containsID(cancelled, 10) {
		t.Fatalf("cancelled = %v, want [10]", cancelled)
	}
}

// The defining CANCEL_NEWEST behaviour: at capacity, a newcomer is rejected even if it is
// higher-priority than a running task. Running work is never preempted (unlike CANCEL_IN_PROGRESS).
func TestCancelNewest_AtCapacityRejectsNewcomerProtectsRunning(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 3, 0, now, future, true), // running, priority 3
			indexRow("a", 2, 3, 0, now, future, true), // running, priority 3
		},
	}
	c := newCancelNewestStrategy(repo, 2)

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	// a higher-priority newcomer arrives at a full group
	msgs := []walMessage{walInsert("a", 3, 9, now, future)}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	// nothing promoted; the newcomer is cancelled; both runners survive
	if got := filledIDs(repo.lastFilled); len(got) != 0 {
		t.Fatalf("filled = %v, want none (running must not be preempted)", got)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 1 || !containsID(cancelled, 3) {
		t.Fatalf("cancelled = %v, want [3]", cancelled)
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
	if _, ok := sq.running.get(3); ok {
		t.Fatalf("rejected newcomer 3 must not be running")
	}
}

// Queued slots past their scheduling timeout are cancelled as SCHEDULING_TIMED_OUT and excluded from
// filling.
func TestCancelNewest_TimedOutExcludedFromFill(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelNewestStrategy(repo, 2)

	msgs := []walMessage{
		walInsert("a", 20, 5, now, future), // valid
		walInsert("a", 21, 9, now, past),   // highest priority, but past timeout
		walInsert("a", 22, 5, now, future), // valid
	}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	timedOut := cancelledByReason(repo.lastCancelled, repository.CancelledReasonSchedulingTimedOut)
	if len(timedOut) != 1 || !containsID(timedOut, 21) {
		t.Fatalf("timed out = %v, want [21]", timedOut)
	}
	filled := filledIDs(repo.lastFilled)
	if len(filled) != 2 || !containsID(filled, 20) || !containsID(filled, 22) {
		t.Fatalf("filled = %v, want {20,22}", filled)
	}
	if got := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit); len(got) != 0 {
		t.Fatalf("unexpected CONCURRENCY_LIMIT cancellations: %v", got)
	}
}

// A retry (higher retry count for the same task) supersedes the older slot, which is cancelled with
// CONCURRENCY_LIMIT, while the retry takes its place.
func TestCancelNewest_RetrySupersedes(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 30, 5, 0, now, future, false), // queued, retry 0
		},
	}
	c := newCancelNewestStrategy(repo, 5) // capacity is not the constraint here

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	retry := walInsert("a", 30, 5, now, future)
	retry.TaskRetryCount = 1
	msgs := []walMessage{retry}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if !containsID(cancelled, 30) {
		t.Fatalf("superseded slot 30 not cancelled: %v", cancelled)
	}
	if got := filledIDs(repo.lastFilled); len(got) != 1 || got[0] != 30 {
		t.Fatalf("filled = %v, want [30] (the retry)", got)
	}

	sq := c.getOrCreateSubQueue("a")
	s, ok := sq.running.get(30)
	if !ok {
		t.Fatalf("retried task 30 should be running")
	}
	if s.taskRetryCount != 1 {
		t.Fatalf("running slot 30 retry count = %d, want 1", s.taskRetryCount)
	}
}

// maxRuns <= 0 leaves no free capacity, so every queued slot is cancelled.
func TestCancelNewest_MaxRunsZeroCancelsAllQueued(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelNewestStrategy(repo, 0)

	msgs := []walMessage{
		walInsert("a", 40, 5, now, future),
		walInsert("a", 41, 5, now, future),
	}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	if got := filledIDs(repo.lastFilled); len(got) != 0 {
		t.Fatalf("filled = %v, want none", got)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	for _, want := range []int64{40, 41} {
		if !containsID(cancelled, want) {
			t.Fatalf("task %d not cancelled; cancelled = %v", want, cancelled)
		}
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.queued.len() != 0 {
		t.Fatalf("queued not drained: %d", sq.queued.len())
	}
}
