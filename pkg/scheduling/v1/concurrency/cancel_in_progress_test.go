package concurrency

import (
	"context"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func newCancelInProgressStrategy(repo repository.ConcurrencyRepository, maxConcurrency int32) *ConcurrencyStrategy {
	return newTestStrategyKind(repo, maxConcurrency, sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS)
}

// At capacity, a higher-priority arrival cancels the lowest-priority running task (in-progress
// cancellation) and is promoted in its place, keeping the best maxRuns under the comparator.
func TestCancelInProgress_NewArrivalCancelsWorstRunning(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 5, 0, now, future, true), // running, priority 5
			indexRow("a", 2, 3, 0, now, future, true), // running, priority 3 (worst)
		},
	}
	c := newCancelInProgressStrategy(repo, 2)

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	// a higher-priority queued slot arrives
	msgs := []walMessage{walInsert("a", 3, 9, now, future)}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	// newcomer (3) promoted; lowest-priority runner (2) cancelled with CONCURRENCY_LIMIT
	if got := filledIDs(repo.lastFilled); len(got) != 1 || got[0] != 3 {
		t.Fatalf("filled = %v, want [3]", got)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 1 || !containsID(cancelled, 2) {
		t.Fatalf("cancelled (CONCURRENCY_LIMIT) = %v, want [2]", cancelled)
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 2 {
		t.Fatalf("running len = %d, want 2", sq.running.len())
	}
	if _, ok := sq.running.get(2); ok {
		t.Fatalf("lowest-priority task 2 should no longer be running")
	}
	if _, ok := sq.running.get(3); !ok {
		t.Fatalf("newcomer 3 should be running")
	}
}

// Keepers are ranked by the comparator (priority, then inserted_at, then taskId): the best maxRuns
// run, the rest are cancelled.
func TestCancelInProgress_RanksByPriority(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelInProgressStrategy(repo, 2)

	msgs := []walMessage{
		walInsert("a", 10, 1, now, future), // lowest priority
		walInsert("a", 11, 5, now, future),
		walInsert("a", 12, 9, now, future), // highest priority
	}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	// two highest priority (12, 11) run; lowest (10) cancelled
	filled := filledIDs(repo.lastFilled)
	if len(filled) != 2 || !containsID(filled, 12) || !containsID(filled, 11) {
		t.Fatalf("filled = %v, want {11,12}", filled)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 1 || !containsID(cancelled, 10) {
		t.Fatalf("cancelled = %v, want [10]", cancelled)
	}
}

// Among equal-priority slots, CANCEL_IN_PROGRESS keeps the NEWEST maxRuns and cancels the oldest
// (matching the legacy runCancelInProgress ORDER BY sort_id DESC). This is the case that actually
// exercises the recency tiebreak - if it were inverted we would keep the oldest instead.
func TestCancelInProgress_KeepsNewestAmongEqualPriority(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelInProgressStrategy(repo, 2)

	// same priority, same inserted_at; recency is distinguished by taskId (higher = newer).
	msgs := []walMessage{
		walInsert("a", 10, 5, now, future), // oldest
		walInsert("a", 11, 5, now, future),
		walInsert("a", 12, 5, now, future), // newest
	}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	// newest two (12, 11) run; oldest (10) cancelled
	filled := filledIDs(repo.lastFilled)
	if len(filled) != 2 || !containsID(filled, 12) || !containsID(filled, 11) {
		t.Fatalf("filled = %v, want {11,12} (newest maxRuns)", filled)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 1 || !containsID(cancelled, 10) {
		t.Fatalf("cancelled = %v, want [10] (oldest)", cancelled)
	}
}

// At capacity with equal priorities, a newer arrival preempts the OLDEST running task (in-progress
// cancellation) - the defining CANCEL_IN_PROGRESS behaviour. Distinct from the priority-driven
// preemption above; this pins the recency direction for running slots.
func TestCancelInProgress_NewerArrivalPreemptsOldestRunning(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 5, 0, now, future, true), // running, oldest
			indexRow("a", 2, 5, 0, now, future, true), // running
		},
	}
	c := newCancelInProgressStrategy(repo, 2)

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	// a newer equal-priority slot arrives at a full group
	msgs := []walMessage{walInsert("a", 3, 5, now, future)}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	// newcomer (3) promoted; oldest runner (1) preempted
	if got := filledIDs(repo.lastFilled); len(got) != 1 || got[0] != 3 {
		t.Fatalf("filled = %v, want [3]", got)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 1 || !containsID(cancelled, 1) {
		t.Fatalf("cancelled = %v, want [1] (oldest running preempted)", cancelled)
	}

	sq := c.getOrCreateSubQueue("a")
	if _, ok := sq.running.get(1); ok {
		t.Fatalf("oldest task 1 should have been preempted")
	}
	if _, ok := sq.running.get(3); !ok {
		t.Fatalf("newcomer 3 should be running")
	}
}

// Queued slots past their scheduling timeout are cancelled as SCHEDULING_TIMED_OUT and excluded from
// the run/cancel ranking, even if they would otherwise outrank the survivors.
func TestCancelInProgress_TimedOutExcludedFromRanking(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newCancelInProgressStrategy(repo, 2)

	msgs := []walMessage{
		walInsert("a", 20, 5, now, future), // valid
		walInsert("a", 21, 9, now, past),   // highest priority, but past timeout
		walInsert("a", 22, 5, now, future), // valid
	}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	// 21 timed out, never ranked; the two valid slots both fit under maxRuns=2 and run
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
func TestCancelInProgress_RetrySupersedes(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 30, 5, 0, now, future, false), // queued, retry 0
		},
	}
	c := newCancelInProgressStrategy(repo, 5) // capacity is not the constraint here

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	// retry of task 30 arrives with retry count 1
	retry := walInsert("a", 30, 5, now, future)
	retry.TaskRetryCount = 1
	msgs := []walMessage{retry}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	// the superseded retry-0 slot is cancelled; the retry-1 slot runs
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

// maxRuns <= 0 keeps nothing: every candidate (running and queued) is cancelled.
func TestCancelInProgress_MaxRunsZeroCancelsAll(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 40, 5, 0, now, future, true),  // running
			indexRow("a", 41, 5, 0, now, future, false), // queued
		},
	}
	c := newCancelInProgressStrategy(repo, 0)

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	msgs := []walMessage{walInsert("a", 42, 5, now, future)}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	if got := filledIDs(repo.lastFilled); len(got) != 0 {
		t.Fatalf("filled = %v, want none", got)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	for _, want := range []int64{40, 41, 42} {
		if !containsID(cancelled, want) {
			t.Fatalf("task %d not cancelled; cancelled = %v", want, cancelled)
		}
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 0 || sq.queued.len() != 0 {
		t.Fatalf("indexes not drained: running %d, queued %d", sq.running.len(), sq.queued.len())
	}
}
