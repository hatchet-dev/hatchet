package concurrency

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func newOnDemandCancelQueuedExceptNewestStrategy(repo repository.ConcurrencyRepository, maxConcurrency int32, eagerIndexMaxSlots int32) *ConcurrencyStrategy {
	c := newTestStrategyKind(repo, maxConcurrency, sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTNEWEST)
	c.eagerIndexMaxSlots = eagerIndexMaxSlots
	return c
}

func TestBuildIndexSwitchesToOnDemandHydrationAtThreshold(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 5, 0, now, future, true),
			indexRow("a", 2, 5, 0, now, future, false),
			indexRow("b", 3, 5, 0, now, future, false),
		},
	}
	c := newOnDemandCancelQueuedExceptNewestStrategy(repo, 1, 3)

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	if !c.hydrateOnDemand {
		t.Fatalf("expected on-demand hydration at the slot threshold")
	}
	if len(c.subQueues) != 0 {
		t.Fatalf("expected no sub-queues hydrated at build, got %d", len(c.subQueues))
	}
}

func TestBuildIndexStaysEagerBelowThreshold(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 5, 0, now, future, true),
			indexRow("a", 2, 5, 0, now, future, false),
		},
	}
	c := newOnDemandCancelQueuedExceptNewestStrategy(repo, 1, 3)

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	if c.hydrateOnDemand {
		t.Fatalf("expected eager hydration below the slot threshold")
	}
	if len(c.subQueues) != 1 {
		t.Fatalf("expected sub-queue a hydrated at build, got %d sub-queues", len(c.subQueues))
	}
}

// Under on-demand hydration a WAL batch loads the touched keys from the database inside the batch,
// decides against those rows rather than the messages, and drops the keys again once committed.
func TestOnDemandWALBatchHydratesDecidesAndEvicts(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 5, 0, now, future, true),                     // running
			indexRow("a", 2, 5, 0, now.Add(time.Second), future, false),   // oldest queued: cancelled
			indexRow("a", 3, 5, 0, now.Add(2*time.Second), future, false), // newest queued: survives
			indexRow("b", 4, 5, 0, now, future, false),                    // untouched key: never loaded
		},
	}
	c := newOnDemandCancelQueuedExceptNewestStrategy(repo, 1, 1)
	c.hydrateOnDemand = true

	// the message's own payload is irrelevant under on-demand hydration: only its key is used
	msgs := []walMessage{walInsert("a", 3, 5, now.Add(2*time.Second), future)}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	if repo.listForKeysCalls != 1 {
		t.Fatalf("expected one hydration query, got %d", repo.listForKeysCalls)
	}
	if len(repo.lastFilled) != 0 {
		t.Fatalf("filled = %v, want none (capacity already taken by running task 1)", filledIDs(repo.lastFilled))
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 1 || !containsID(cancelled, 2) {
		t.Fatalf("cancelled = %v, want [2]", cancelled)
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 1 || sq.queued.len() != 1 {
		t.Fatalf("before commit: running %d, queued %d, want 1, 1", sq.running.len(), sq.queued.len())
	}
	if _, hasB := c.subQueues["b"]; hasB {
		t.Fatalf("key b was hydrated although no message touched it")
	}

	c.finalizeCommittedSubQueues(c.commitScopes())

	if len(c.subQueues) != 0 {
		t.Fatalf("expected every touched sub-queue evicted after commit, got %d", len(c.subQueues))
	}
}

func TestOnDemandWALBatchEvictsOnRollback(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 5, 0, now, future, false),
		},
		updateErr: errors.New("db unavailable"),
	}
	c := newOnDemandCancelQueuedExceptNewestStrategy(repo, 1, 1)
	c.hydrateOnDemand = true

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{walInsert("a", 1, 5, now, future)}); err == nil {
		t.Fatalf("expected error from failed flush")
	}

	c.rollbackScopes()

	if len(c.subQueues) != 0 {
		t.Fatalf("expected touched sub-queues evicted after rollback, got %d", len(c.subQueues))
	}
}

// The initial queueing pass under on-demand hydration walks the key space a page at a time across
// Runs, flushing each page in its own transaction and evicting it, and only reports done once the
// scan runs off the end of the keys.
func TestOnDemandInitialQueueingScansKeysAcrossRuns(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	keyCount := onDemandInitialScanKeysPerPage*onDemandInitialScanPagesPerRun + 1

	rows := make([]*sqlcv1.ListConcurrencySlotsForIndexingRow, 0, 2*keyCount)
	for i := range keyCount {
		key := fmt.Sprintf("key-%06d", i)
		rows = append(rows,
			indexRow(key, int64(2*i+1), 5, 0, now, future, false),
			indexRow(key, int64(2*i+2), 5, 0, now.Add(time.Second), future, false),
		)
	}

	repo := &mockConcurrencyRepo{indexRows: rows}
	c := newOnDemandCancelQueuedExceptNewestStrategy(repo, 1, 1)
	c.hydrateOnDemand = true

	// first Run: a full set of pages, every key decided and flushed, scan not yet complete
	res, err := c.runInitialQueueing(context.Background())
	if err != nil {
		t.Fatalf("runInitialQueueing (first): %v", err)
	}
	if res == nil {
		t.Fatalf("nil result from first scan pass")
	}
	if c.initialQueued {
		t.Fatalf("scan reported complete after the first pass")
	}
	if repo.updateCalls != onDemandInitialScanPagesPerRun {
		t.Fatalf("update calls after first pass = %d, want %d", repo.updateCalls, onDemandInitialScanPagesPerRun)
	}
	if len(c.subQueues) != 0 {
		t.Fatalf("expected scanned pages evicted, got %d sub-queues", len(c.subQueues))
	}

	// per key: the older queued slot fills the single free slot, the newer one survives queued
	filled := filledIDs(repo.lastFilled)
	if len(filled) != onDemandInitialScanKeysPerPage {
		t.Fatalf("last page filled %d slots, want %d", len(filled), onDemandInitialScanKeysPerPage)
	}
	if len(repo.lastCancelled) != 0 {
		t.Fatalf("last page cancelled %d slots, want 0", len(repo.lastCancelled))
	}

	// second Run: the single remaining key, then an empty page marks the scan complete
	if _, err := c.runInitialQueueing(context.Background()); err != nil {
		t.Fatalf("runInitialQueueing (second): %v", err)
	}
	if !c.initialQueued {
		t.Fatalf("scan not reported complete after exhausting the keys")
	}
	if repo.updateCalls != onDemandInitialScanPagesPerRun+1 {
		t.Fatalf("update calls after second pass = %d, want %d", repo.updateCalls, onDemandInitialScanPagesPerRun+1)
	}

	// third Run: no-op
	if res, err := c.runInitialQueueing(context.Background()); err != nil || res != nil {
		t.Fatalf("expected no-op (nil, nil), got (%v, %v)", res, err)
	}
}

func TestOnDemandInitialQueueingResumesFromCursorAfterFailure(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	rows := make([]*sqlcv1.ListConcurrencySlotsForIndexingRow, 0)
	for i := range 2 * onDemandInitialScanKeysPerPage {
		rows = append(rows, indexRow(fmt.Sprintf("key-%06d", i), int64(i+1), 5, 0, now, future, false))
	}

	repo := &mockConcurrencyRepo{indexRows: rows, updateErr: errors.New("db unavailable")}
	c := newOnDemandCancelQueuedExceptNewestStrategy(repo, 1, 1)
	c.hydrateOnDemand = true

	if _, err := c.runInitialQueueing(context.Background()); err == nil {
		t.Fatalf("expected error from failed flush")
	}
	if c.initialQueued {
		t.Fatalf("scan marked complete despite flush failure")
	}
	if c.initialScanLastKey != "" {
		t.Fatalf("cursor advanced past a page whose flush failed: %q", c.initialScanLastKey)
	}
	if len(c.subQueues) != 0 {
		t.Fatalf("expected failed page evicted, got %d sub-queues", len(c.subQueues))
	}

	repo.updateErr = nil

	if _, err := c.runInitialQueueing(context.Background()); err != nil {
		t.Fatalf("runInitialQueueing retry: %v", err)
	}
	if !c.initialQueued {
		t.Fatalf("scan not complete after retry")
	}
}
