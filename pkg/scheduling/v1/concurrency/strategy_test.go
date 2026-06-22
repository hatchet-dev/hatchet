package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// mockConcurrencyRepo is an in-memory ConcurrencyRepository: ReadConcurrencySlotsForIndexing
// replays a fixed set of rows and UpdateConcurrencySlots captures its inputs for assertions.
// The pgx.Tx handed to UpdateConcurrencySlots is opaque here, so tests pass a nil tx.
type mockConcurrencyRepo struct {
	indexRows []*sqlcv1.ListConcurrencySlotsForIndexingRow

	updateResult *repository.RunConcurrencyResult
	updateErr    error

	// captured from the most recent UpdateConcurrencySlots call
	lastFilled    []repository.TaskIdInsertedAtRetryCount
	lastCancelled []repository.CancelledSlotInput
	updateCalls   int
}

func (m *mockConcurrencyRepo) ReadConcurrencySlotsForIndexing(ctx context.Context, tenantId uuid.UUID, strategyId int64, writeCh chan<- *sqlcv1.ListConcurrencySlotsForIndexingRow) error {
	for _, r := range m.indexRows {
		writeCh <- r
	}
	return nil
}

func (m *mockConcurrencyRepo) UpdateConcurrencySlotsTx(ctx context.Context, tx pgx.Tx, tenantId uuid.UUID, strategyId int64, filledSlots []repository.TaskIdInsertedAtRetryCount, cancelledSlots []repository.CancelledSlotInput) (*repository.RunConcurrencyResult, error) {
	m.updateCalls++
	m.lastFilled = filledSlots
	m.lastCancelled = cancelledSlots
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if m.updateResult != nil {
		return m.updateResult, nil
	}
	return &repository.RunConcurrencyResult{}, nil
}

func (m *mockConcurrencyRepo) UpdateConcurrencySlots(ctx context.Context, tenantId uuid.UUID, strategyId int64, filledSlots []repository.TaskIdInsertedAtRetryCount, cancelledSlots []repository.CancelledSlotInput) (*repository.RunConcurrencyResult, error) {
	return m.UpdateConcurrencySlotsTx(ctx, nil, tenantId, strategyId, filledSlots, cancelledSlots)
}

// remaining interface methods are unused by these tests.
func (m *mockConcurrencyRepo) UpdateConcurrencyStrategyIsActive(ctx context.Context, tenantId uuid.UUID, strategy *sqlcv1.V1StepConcurrency) error {
	return nil
}
func (m *mockConcurrencyRepo) RunConcurrencyStrategy(ctx context.Context, tenantId uuid.UUID, strategy *sqlcv1.V1StepConcurrency) (*repository.RunConcurrencyResult, error) {
	return nil, nil
}
func (m *mockConcurrencyRepo) DeactivateStaleStepConcurrency(ctx context.Context, tenantId uuid.UUID) error {
	return nil
}
func (m *mockConcurrencyRepo) ListTenantsWithManyStepConcurrencies(ctx context.Context, threshold int64) ([]*sqlcv1.ListTenantsWithManyStepConcurrenciesRow, error) {
	return nil, nil
}

func newTestStrategy(repo repository.ConcurrencyRepository, maxConcurrency int32) *ConcurrencyStrategy {
	return newTestStrategyKind(repo, maxConcurrency, sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN)
}

func newTestStrategyKind(repo repository.ConcurrencyRepository, maxConcurrency int32, kind sqlcv1.V1ConcurrencyStrategy) *ConcurrencyStrategy {
	l := zerolog.Nop()
	return &ConcurrencyStrategy{
		subQueues: make(map[string]*subQueue),
		strategy:  &sqlcv1.V1StepConcurrency{MaxConcurrency: maxConcurrency, Strategy: kind},
		repo:      repo,
		l:         &l,
		compare:   comparatorForStrategy(kind),
		built:     make(chan struct{}),
	}
}

func indexRow(key string, taskId int64, priority, retry int32, insertedAt, timeoutAt time.Time, filled bool) *sqlcv1.ListConcurrencySlotsForIndexingRow {
	return &sqlcv1.ListConcurrencySlotsForIndexingRow{
		TaskID:            taskId,
		TaskInsertedAt:    pgtype.Timestamptz{Time: insertedAt, Valid: true},
		TaskRetryCount:    retry,
		Key:               key,
		Priority:          priority,
		IsFilled:          filled,
		ScheduleTimeoutAt: pgtype.Timestamp{Time: timeoutAt, Valid: true},
	}
}

func walInsert(key string, taskId int64, priority int32, insertedAt, timeoutAt time.Time) walMessage {
	return walMessage{
		Operation:           "INSERT",
		Key:                 key,
		Priority:            priority,
		TaskId:              taskId,
		TaskInsertedAt:      insertedAt,
		ScheduleTimeoutAtMs: timeoutAt.UnixMilli(),
	}
}

func filledIDs(filled []repository.TaskIdInsertedAtRetryCount) []int64 {
	ids := make([]int64, len(filled))
	for i, f := range filled {
		ids[i] = f.Id
	}
	return ids
}

// cancelledByReason returns the taskIds cancelled with the given reason.
func cancelledByReason(cancelled []repository.CancelledSlotInput, reason string) []int64 {
	var ids []int64
	for _, c := range cancelled {
		if c.CancelledReason == reason {
			ids = append(ids, c.Id)
		}
	}
	return ids
}

func containsID(ids []int64, want int64) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func TestBuildIndexHydratesSubQueues(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 5, 0, now, future, false), // queued
			indexRow("a", 2, 5, 0, now, future, false), // queued
			indexRow("a", 3, 5, 0, now, future, true),  // running
			indexRow("b", 4, 5, 0, now, future, false), // queued
		},
	}
	c := newTestStrategy(repo, 5)

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	sqA := c.getOrCreateSubQueue("a")
	if sqA.queued.len() != 2 {
		t.Fatalf("subQueue a queued len = %d, want 2", sqA.queued.len())
	}
	if sqA.running.len() != 1 {
		t.Fatalf("subQueue a running len = %d, want 1", sqA.running.len())
	}
	if _, ok := sqA.running.get(3); !ok {
		t.Fatalf("task 3 not found in running index")
	}

	sqB := c.getOrCreateSubQueue("b")
	if sqB.queued.len() != 1 || sqB.running.len() != 0 {
		t.Fatalf("subQueue b = (queued %d, running %d), want (1, 0)", sqB.queued.len(), sqB.running.len())
	}
}

func TestProcessDequeuesByPriority(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newTestStrategy(repo, 2) // only 2 slots may run

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

	// mirror Run's success path
	c.commitScopes()
	if c.openScopes != nil {
		t.Fatalf("openScopes not cleared after commit")
	}
}

func TestProcessCancelsTimedOutBeforeFilling(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newTestStrategy(repo, 5) // plenty of room; timeout, not capacity, is under test

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

func TestProcessRollbackOnFlushError(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{updateErr: errors.New("db unavailable")}
	c := newTestStrategy(repo, 2)

	msgs := []walMessage{
		walInsert("a", 301, 1, now, future),
		walInsert("a", 305, 5, now, future),
	}

	_, err := c.processWALMessages(context.Background(), nil, msgs)
	if err == nil {
		t.Fatalf("expected error from failed flush")
	}

	// Run rolls back the open scopes when the transaction fails; the index must return to its
	// pre-batch (empty) state so it stays consistent with the un-committed database.
	c.rollbackScopes()

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 0 || sq.queued.len() != 0 {
		t.Fatalf("after rollback: running %d, queued %d, want 0, 0", sq.running.len(), sq.queued.len())
	}
	if c.openScopes != nil {
		t.Fatalf("openScopes not cleared after rollback")
	}
}

// TestQueueAllSubQueuesAfterBuild covers the post-build queueing pass: buildIndex hydrates queued
// backlog but never runs the decide step, so the first pass must promote that backlog into free
// capacity and flush it, across every sub-queue, without any WAL message arriving.
func TestQueueAllSubQueuesAfterBuild(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 5, 0, now, future, false), // queued
			indexRow("a", 2, 9, 0, now, future, false), // queued, higher priority
			indexRow("b", 3, 5, 0, now, future, false), // queued, different sub-queue
		},
	}
	c := newTestStrategy(repo, 5) // ample capacity: all backlog should be promoted

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	res, err := c.queueAllSubQueues(context.Background())
	if err != nil {
		t.Fatalf("queueAllSubQueues: %v", err)
	}
	if res == nil {
		t.Fatalf("nil result")
	}

	// every hydrated queued slot is promoted to running and flushed as filled, even though no WAL
	// message ever touched its sub-queue.
	filled := filledIDs(repo.lastFilled)
	for _, want := range []int64{1, 2, 3} {
		if !containsID(filled, want) {
			t.Fatalf("task %d not filled by initial queueing: %v", want, filled)
		}
	}

	sqA := c.getOrCreateSubQueue("a")
	if sqA.running.len() != 2 || sqA.queued.len() != 0 {
		t.Fatalf("sub-queue a = (running %d, queued %d), want (2, 0)", sqA.running.len(), sqA.queued.len())
	}
}

// TestRunInitialQueueingOnce verifies the post-build pass runs exactly once on success and only
// retries after a failure.
func TestRunInitialQueueingOnce(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 5, 0, now, future, false),
		},
		updateErr: errors.New("db unavailable"),
	}
	c := newTestStrategy(repo, 5)

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	// first attempt fails: the pass must not mark itself done, leaving it to retry.
	if _, err := c.runInitialQueueing(context.Background()); err == nil {
		t.Fatalf("expected error from failed initial queueing")
	}
	if c.initialQueued {
		t.Fatalf("initialQueued set despite flush failure")
	}

	// recover and retry: succeeds and marks done.
	repo.updateErr = nil
	if _, err := c.runInitialQueueing(context.Background()); err != nil {
		t.Fatalf("runInitialQueueing retry: %v", err)
	}
	if !c.initialQueued {
		t.Fatalf("initialQueued not set after success")
	}

	callsAfterSuccess := repo.updateCalls

	// subsequent calls are no-ops: no further flush.
	if res, err := c.runInitialQueueing(context.Background()); err != nil || res != nil {
		t.Fatalf("expected no-op (nil, nil), got (%v, %v)", res, err)
	}
	if repo.updateCalls != callsAfterSuccess {
		t.Fatalf("initial queueing flushed again: calls %d, want %d", repo.updateCalls, callsAfterSuccess)
	}
}

// TestPruneEmptyAfterCommit verifies that a sub-queue emptied by a batch (all slots deleted) is
// removed from the index, while a sub-queue that still holds slots is retained.
func TestPruneEmptyAfterCommit(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newTestStrategy(repo, 5)

	// seed sub-queue "a" with a single running slot, then delete it; "b" keeps a slot.
	insertMsgs := []walMessage{
		walInsert("a", 1, 5, now, future),
		walInsert("b", 2, 5, now, future),
	}
	if _, err := c.processWALMessages(context.Background(), nil, insertMsgs); err != nil {
		t.Fatalf("processWALMessages (insert): %v", err)
	}
	c.pruneEmpty(c.commitScopes())

	// now delete task 1, emptying sub-queue "a".
	deleteMsgs := []walMessage{{Operation: "DELETE", Key: "a", TaskId: 1}}
	if _, err := c.processWALMessages(context.Background(), nil, deleteMsgs); err != nil {
		t.Fatalf("processWALMessages (delete): %v", err)
	}
	c.pruneEmpty(c.commitScopes())

	c.mu.RLock()
	_, hasA := c.subQueues["a"]
	_, hasB := c.subQueues["b"]
	c.mu.RUnlock()

	if hasA {
		t.Fatalf("empty sub-queue a was not pruned")
	}
	if !hasB {
		t.Fatalf("non-empty sub-queue b was incorrectly pruned")
	}
}
