package concurrency

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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

func TestOnDemandInitialQueueingIncludesEmptyKey(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("", 1, 5, 0, now, future, false),
			indexRow("a", 2, 5, 0, now, future, false),
		},
	}
	c := newOnDemandCancelQueuedExceptNewestStrategy(repo, 1, 1)
	c.hydrateOnDemand = true

	if _, err := c.runInitialQueueing(context.Background()); err != nil {
		t.Fatalf("runInitialQueueing: %v", err)
	}

	filled := filledIDs(repo.lastFilled)
	if len(filled) != 2 || !containsID(filled, 1) {
		t.Fatalf("filled = %v, want the empty-key task 1 and task 2", filled)
	}
}

// A page failing after earlier pages committed must not drop those pages' results: the cancelled
// tasks' messages are published from them. The scan resumes from the cursor on the next Run.
func TestOnDemandInitialQueueingKeepsCommittedResultsWhenLaterPageFails(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	rows := make([]*sqlcv1.ListConcurrencySlotsForIndexingRow, 0)
	for i := range 3 * onDemandInitialScanKeysPerPage {
		key := fmt.Sprintf("key-%06d", i)
		rows = append(rows,
			indexRow(key, int64(2*i+1), 5, 0, now, future, true),
			indexRow(key, int64(2*i+2), 5, 0, now.Add(time.Second), future, false),
			indexRow(key, int64(2*i+3), 5, 0, now.Add(2*time.Second), future, false),
		)
	}

	cancelledPerPage := onDemandInitialScanKeysPerPage
	repo := &mockConcurrencyRepo{
		indexRows:           rows,
		updateErr:           errors.New("db unavailable"),
		updateSucceedsFirst: 2,
		updateResult: &repository.RunConcurrencyResult{
			Cancelled: make([]repository.TaskWithCancelledReason, cancelledPerPage),
		},
	}
	c := newOnDemandCancelQueuedExceptNewestStrategy(repo, 1, 1)
	c.hydrateOnDemand = true

	res, err := c.runInitialQueueing(context.Background())
	if err != nil {
		t.Fatalf("expected the committed pages' results, got error: %v", err)
	}
	if len(res.Cancelled) != 2*cancelledPerPage {
		t.Fatalf("cancelled results = %d, want %d from the two committed pages", len(res.Cancelled), 2*cancelledPerPage)
	}
	if c.initialQueued {
		t.Fatalf("scan marked complete although the third page failed")
	}
	if !c.initialScanLastKey.Valid || c.initialScanLastKey.String != fmt.Sprintf("key-%06d", 2*onDemandInitialScanKeysPerPage-1) {
		t.Fatalf("cursor = %q, want the last key of the second page", c.initialScanLastKey.String)
	}
}

// An eagerly hydrated index that grows past the bound after build switches to on-demand mode,
// dropping its sub-queues and re-arming the paged scan.
func TestEagerIndexSwitchesToOnDemandWhenItOutgrowsBound(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newOnDemandCancelQueuedExceptNewestStrategy(repo, 1, 3)
	c.initialQueued = true

	msgs := []walMessage{
		walInsert("a", 1, 5, now, future),
		walInsert("b", 2, 5, now, future),
	}
	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.finalizeCommittedSubQueues(c.commitScopes())

	if c.eagerIndexOutgrewBound() {
		t.Fatalf("two keys should be under a bound of three")
	}

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{walInsert("c", 3, 5, now, future)}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.finalizeCommittedSubQueues(c.commitScopes())

	if !c.eagerIndexOutgrewBound() {
		t.Fatalf("three keys should reach a bound of three")
	}

	c.switchToOnDemandHydration()

	if !c.hydrateOnDemand {
		t.Fatalf("expected on-demand hydration after the switch")
	}
	if len(c.subQueues) != 0 {
		t.Fatalf("expected hydrated sub-queues dropped, got %d", len(c.subQueues))
	}
	if c.initialQueued || c.initialScanLastKey.Valid {
		t.Fatalf("expected the initial scan re-armed from the start")
	}
	if c.eagerIndexOutgrewBound() {
		t.Fatalf("bound check must be inert once on demand")
	}
}

// A deep key is hydrated through a bounded window: only the best and worst maxRuns queued slots are
// loaded, and the queued slots outside that window are cancelled straight from the rows.
func TestOnDemandDeepKeyLoadsBoundedWindowAndCancelsOutside(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	queuedCount := 50
	rows := make([]*sqlcv1.ListConcurrencySlotsForIndexingRow, 0, queuedCount)
	for i := range queuedCount {
		rows = append(rows, indexRow("a", int64(i+1), 5, 0, now.Add(time.Duration(i)*time.Second), future, false))
	}

	repo := &mockConcurrencyRepo{indexRows: rows}
	c := newOnDemandCancelQueuedExceptNewestStrategy(repo, 1, 1)
	c.hydrateOnDemand = true

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{walInsert("a", 50, 5, now, future)}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	if got := repo.lastWindowQuery.Limits; len(got) != 1 || got[0] != 1 || repo.lastWindowQuery.LimitFactor != 1 {
		t.Fatalf("limits = %v, factor = %d, want [1] and 1", got, repo.lastWindowQuery.LimitFactor)
	}
	if repo.lastWindowQuery.OutsideLimit != onDemandOutsideWindowCancelLimit {
		t.Fatalf("outside limit = %d, want %d", repo.lastWindowQuery.OutsideLimit, onDemandOutsideWindowCancelLimit)
	}

	// oldest fills the slot, newest survives queued, the 48 in between are cancelled
	filled := filledIDs(repo.lastFilled)
	if len(filled) != 1 || !containsID(filled, 1) {
		t.Fatalf("filled = %v, want [1]", filled)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != queuedCount-2 {
		t.Fatalf("cancelled %d slots, want %d", len(cancelled), queuedCount-2)
	}
	if containsID(cancelled, 1) || containsID(cancelled, 50) {
		t.Fatalf("cancelled the filled or the surviving slot: %v", cancelled)
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 1 || sq.queued.len() != 1 {
		t.Fatalf("in-memory running %d, queued %d, want 1, 1: the window must hold two queued slots at most", sq.running.len(), sq.queued.len())
	}
	if len(c.keysToRevisit) != 0 {
		t.Fatalf("no revisit expected when the outside cap was not reached")
	}
}

func TestOnDemandGroupRoundRobinLeavesOutsideWindowQueued(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	rows := make([]*sqlcv1.ListConcurrencySlotsForIndexingRow, 0)
	for i := range 20 {
		rows = append(rows, indexRow("a", int64(i+1), 5, 0, now.Add(time.Duration(i)*time.Second), future, false))
	}

	repo := &mockConcurrencyRepo{indexRows: rows}
	c := newTestStrategyKind(repo, 2, sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN)
	c.hydrateOnDemand = true

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{walInsert("a", 20, 5, now, future)}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	if repo.lastWindowQuery.OutsideLimit != 0 {
		t.Fatalf("outside limit = %d, want 0: round robin never cancels queued backlog", repo.lastWindowQuery.OutsideLimit)
	}
	filled := filledIDs(repo.lastFilled)
	if len(filled) != 2 || !containsID(filled, 1) || !containsID(filled, 2) {
		t.Fatalf("filled = %v, want the two oldest [1 2]", filled)
	}
	if len(repo.lastCancelled) != 0 {
		t.Fatalf("cancelled %d slots, want 0", len(repo.lastCancelled))
	}
}

func TestOnDemandCancelQueuedExceptOldestDoublesWindow(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	rows := make([]*sqlcv1.ListConcurrencySlotsForIndexingRow, 0)
	for i := range 10 {
		rows = append(rows, indexRow("a", int64(i+1), 5, 0, now.Add(time.Duration(i)*time.Second), future, false))
	}

	repo := &mockConcurrencyRepo{indexRows: rows}
	c := newTestStrategyKind(repo, 2, sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTOLDEST)
	c.hydrateOnDemand = true

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{walInsert("a", 10, 5, now, future)}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	if got := repo.lastWindowQuery.Limits; len(got) != 1 || got[0] != 2 || repo.lastWindowQuery.LimitFactor != 2 {
		t.Fatalf("limits = %v, factor = %d, want [2] and 2 (twice the limit)", got, repo.lastWindowQuery.LimitFactor)
	}

	// the two oldest fill, the next two oldest stay queued, the remaining six are cancelled
	filled := filledIDs(repo.lastFilled)
	if len(filled) != 2 || !containsID(filled, 1) || !containsID(filled, 2) {
		t.Fatalf("filled = %v, want [1 2]", filled)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 6 || containsID(cancelled, 3) || containsID(cancelled, 4) {
		t.Fatalf("cancelled = %v, want the six newest", cancelled)
	}
}

// When a batch's outside-window cancellations hit the cap, its keys are revisited on the next Run so
// the rest of the backlog is cancelled in further bounded passes.
func TestOnDemandCappedOutsideCancelsAreRevisited(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	// 3 keys x (cap + 5) queued: the first batch can cancel at most cap slots across the keys
	cap := onDemandOutsideWindowCancelLimit
	rows := make([]*sqlcv1.ListConcurrencySlotsForIndexingRow, 0)
	taskId := int64(0)
	for _, key := range []string{"a", "b", "c"} {
		for i := range cap + 5 {
			taskId++
			rows = append(rows, indexRow(key, taskId, 5, 0, now.Add(time.Duration(i)*time.Millisecond), future, false))
		}
	}

	repo := &mockConcurrencyRepo{indexRows: rows}
	c := newOnDemandCancelQueuedExceptNewestStrategy(repo, 1, 1)
	c.hydrateOnDemand = true
	c.initialQueued = true

	msgs := []walMessage{walInsert("a", 1, 5, now, future), walInsert("b", 2, 5, now, future), walInsert("c", 3, 5, now, future)}
	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.finalizeCommittedSubQueues(c.commitScopes())

	if got := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit); len(got) != cap {
		t.Fatalf("first batch cancelled %d, want the cap %d", len(got), cap)
	}
	if len(c.keysToRevisit) != 3 {
		t.Fatalf("keys to revisit = %d, want all 3 touched keys", len(c.keysToRevisit))
	}

	// the mock does not delete rows, so a revisit sees the same backlog and is capped again: the keys
	// must re-enter the set rather than be dropped
	res, err := c.revisitKeys(context.Background())
	if err != nil {
		t.Fatalf("revisitKeys: %v", err)
	}
	if res == nil {
		t.Fatalf("expected results from the revisit pass")
	}
	if len(c.keysToRevisit) != 3 {
		t.Fatalf("keys to revisit after a still-capped pass = %d, want 3", len(c.keysToRevisit))
	}
	if len(c.subQueues) != 0 {
		t.Fatalf("revisited keys must be evicted, got %d sub-queues", len(c.subQueues))
	}
}

// A dynamic limit observed from the newest task must survive that task's slot disappearing between
// batches: on reload, the older slots' higher evaluation must not win back the limit.
func TestOnDemandDynamicLimitSurvivesEviction(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	withMaxRuns := func(row *sqlcv1.ListConcurrencySlotsForIndexingRow, maxRuns int32) *sqlcv1.ListConcurrencySlotsForIndexingRow {
		row.MaxRuns = pgtype.Int4{Int32: maxRuns, Valid: true}
		return row
	}

	running := withMaxRuns(indexRow("a", 1, 5, 0, now, future, true), 3)
	olderQueued := withMaxRuns(indexRow("a", 2, 5, 0, now.Add(time.Second), future, false), 3)
	newerQueued := withMaxRuns(indexRow("a", 3, 5, 0, now.Add(2*time.Second), future, false), 1)

	repo := &mockConcurrencyRepo{indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{running, olderQueued, newerQueued}}
	c := newTestStrategyKind(repo, 5, sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN)
	c.strategy.MaxRunsExpression = pgtype.Text{String: "input.limit", Valid: true}
	c.hydrateOnDemand = true

	// newest task says the limit is 1 and one slot is already running: nothing fills
	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{walInsert("a", 3, 5, now.Add(2*time.Second), future)}); err != nil {
		t.Fatalf("processWALMessages (first): %v", err)
	}
	c.finalizeCommittedSubQueues(c.commitScopes())

	if len(repo.lastFilled) != 0 {
		t.Fatalf("filled %v with limit 1 and one running", filledIDs(repo.lastFilled))
	}
	if observed, ok := c.observedMaxRuns["a"]; !ok || observed.maxRuns != 1 {
		t.Fatalf("observed limit = %+v, want 1 retained across eviction", observed)
	}

	// the newest task's slot is gone (finished or timed out); only the older slots remain
	repo.indexRows = []*sqlcv1.ListConcurrencySlotsForIndexingRow{running, olderQueued}

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{{Operation: "DELETE", Key: "a", TaskId: 3}}); err != nil {
		t.Fatalf("processWALMessages (second): %v", err)
	}
	c.finalizeCommittedSubQueues(c.commitScopes())

	if len(repo.lastFilled) != 0 {
		t.Fatalf("filled %v: a deleted newer task must not raise the limit back to 3", filledIDs(repo.lastFilled))
	}

	// once the key holds no slots the observation is forgotten, as pruneEmpty does on the eager path
	repo.indexRows = nil

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{{Operation: "DELETE", Key: "a", TaskId: 1}}); err != nil {
		t.Fatalf("processWALMessages (third): %v", err)
	}
	c.finalizeCommittedSubQueues(c.commitScopes())

	if _, ok := c.observedMaxRuns["a"]; ok {
		t.Fatalf("observation retained for a key with no slots")
	}
}

// An expired queued slot must not take a window position: CANCEL_NEWEST with limit 1, an expired
// high-priority task, a valid middle-priority task and a valid low-priority task must run the middle
// one, cancel the low one with CONCURRENCY_LIMIT and time out the expired one.
func TestOnDemandExpiredSlotsDoNotDisplaceValidWork(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Minute)
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 9, 0, now, past, false),   // expired, highest priority
			indexRow("a", 2, 5, 0, now, future, false), // valid, middle priority: should run
			indexRow("a", 3, 1, 0, now, future, false), // valid, lowest priority: cancelled
		},
	}
	c := newTestStrategyKind(repo, 1, sqlcv1.V1ConcurrencyStrategyCANCELNEWEST)
	c.hydrateOnDemand = true

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{walInsert("a", 3, 1, now, future)}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	filled := filledIDs(repo.lastFilled)
	if len(filled) != 1 || !containsID(filled, 2) {
		t.Fatalf("filled = %v, want [2]", filled)
	}
	limited := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(limited) != 1 || !containsID(limited, 3) {
		t.Fatalf("CONCURRENCY_LIMIT cancels = %v, want [3]", limited)
	}
	timedOut := cancelledByReason(repo.lastCancelled, repository.CancelledReasonSchedulingTimedOut)
	if len(timedOut) != 1 || !containsID(timedOut, 1) {
		t.Fatalf("SCHEDULING_TIMED_OUT cancels = %v, want [1]", timedOut)
	}
}

// On a dynamic strategy the first visit of a key knows only the static limit; the window must still be
// sized by the limit the key's own slots evaluated, or valid work gets cancelled.
func TestOnDemandWindowWidensToEvaluatedLimitOnFirstVisit(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	rows := make([]*sqlcv1.ListConcurrencySlotsForIndexingRow, 0, 5)
	for i := range 5 {
		row := indexRow("a", int64(i+1), 5, 0, now.Add(time.Duration(i)*time.Second), future, false)
		row.MaxRuns = pgtype.Int4{Int32: 3, Valid: true}
		rows = append(rows, row)
	}

	repo := &mockConcurrencyRepo{indexRows: rows}
	c := newTestStrategyKind(repo, 1, sqlcv1.V1ConcurrencyStrategyCANCELNEWEST)
	c.strategy.MaxRunsExpression = pgtype.Text{String: "input.limit", Valid: true}
	c.hydrateOnDemand = true

	if _, err := c.runInitialQueueing(context.Background()); err != nil {
		t.Fatalf("runInitialQueueing: %v", err)
	}

	filled := filledIDs(repo.lastFilled)
	if len(filled) != 3 || !containsID(filled, 1) || !containsID(filled, 2) || !containsID(filled, 3) {
		t.Fatalf("filled = %v, want the three oldest under the evaluated limit of 3", filled)
	}
	cancelled := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit)
	if len(cancelled) != 2 {
		t.Fatalf("cancelled = %v, want the two newest", cancelled)
	}
}

// Slots created together share a timestamp. An older expired slot that evaluated a higher limit must not
// overwrite the newest slot's lower limit just because expired rows are loaded last.
func TestOnDemandExpiredSlotWithEqualTimestampDoesNotRaiseLimit(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Minute)
	future := now.Add(time.Hour)

	withMaxRuns := func(row *sqlcv1.ListConcurrencySlotsForIndexingRow, maxRuns int32) *sqlcv1.ListConcurrencySlotsForIndexingRow {
		row.MaxRuns = pgtype.Int4{Int32: maxRuns, Valid: true}
		return row
	}

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			withMaxRuns(indexRow("a", 1, 5, 0, now, future, true), 3),  // running
			withMaxRuns(indexRow("a", 2, 5, 0, now, past, false), 3),   // expired, same timestamp as the newest
			withMaxRuns(indexRow("a", 3, 5, 0, now, future, false), 1), // newest by task id: limit 1
			withMaxRuns(indexRow("a", 4, 5, 0, now.Add(-time.Second), future, false), 3),
		},
	}
	c := newTestStrategyKind(repo, 1, sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN)
	c.strategy.MaxRunsExpression = pgtype.Text{String: "input.limit", Valid: true}
	c.hydrateOnDemand = true

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{{Operation: "DELETE", Key: "a", TaskId: 99}}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	// limit 1 with one running slot: nothing fills, the expired slot times out
	if len(repo.lastFilled) != 0 {
		t.Fatalf("filled %v: the expired slot's limit of 3 overwrote the newest slot's limit of 1", filledIDs(repo.lastFilled))
	}
	timedOut := cancelledByReason(repo.lastCancelled, repository.CancelledReasonSchedulingTimedOut)
	if len(timedOut) != 1 || !containsID(timedOut, 2) {
		t.Fatalf("SCHEDULING_TIMED_OUT cancels = %v, want [2]", timedOut)
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
	if c.initialScanLastKey.Valid {
		t.Fatalf("cursor advanced past a page whose flush failed: %q", c.initialScanLastKey.String)
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
