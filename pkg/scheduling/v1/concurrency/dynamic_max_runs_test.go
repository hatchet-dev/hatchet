package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func walInsertMaxRuns(key string, taskId int64, priority int32, insertedAt, timeoutAt time.Time, maxRuns int32) walMessage {
	m := walInsert(key, taskId, priority, insertedAt, timeoutAt)
	m.MaxRuns = &maxRuns
	return m
}

func indexRowMaxRuns(key string, taskId int64, priority, retry int32, insertedAt, timeoutAt time.Time, filled bool, maxRuns int32) *sqlcv1.ListConcurrencySlotsForIndexingRow {
	r := indexRow(key, taskId, priority, retry, insertedAt, timeoutAt, filled)
	r.MaxRuns = pgtype.Int4{Int32: maxRuns, Valid: true}
	return r
}

// Two groups on the same strategy get different limits from their slots' evaluated
// max-runs values: without the dynamic value both groups would be capped at the static
// MaxConcurrency of 1.
func TestDynamicMaxRuns_PerGroupLimits(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newGroupRoundRobinStrategy(repo, 1) // static limit 1: the dynamic values must override it

	msgs := []walMessage{
		walInsertMaxRuns("premium", 101, 1, now, future, 3),
		walInsertMaxRuns("premium", 102, 1, now, future, 3),
		walInsertMaxRuns("premium", 103, 1, now, future, 3),
		walInsertMaxRuns("free", 201, 1, now, future, 1),
		walInsertMaxRuns("free", 202, 1, now, future, 1),
	}

	if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}

	sqPremium := c.getOrCreateSubQueue("premium")
	if sqPremium.running.len() != 3 {
		t.Fatalf("premium running = %d, want 3 (dynamic limit must override static 1)", sqPremium.running.len())
	}

	sqFree := c.getOrCreateSubQueue("free")
	if sqFree.running.len() != 1 || sqFree.queued.len() != 1 {
		t.Fatalf("free = (running %d, queued %d), want (1, 1)", sqFree.running.len(), sqFree.queued.len())
	}
}

// A slot re-inserted for an older task (replay, retry requeue) carries the original task
// timestamp, so its evaluated value must not regress a limit set by a newer task.
func TestDynamicMaxRuns_ReplayOfOlderTaskCannotRegressLimit(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newGroupRoundRobinStrategy(repo, 1)

	// older task evaluated 5, newer task evaluated 2 -> effective limit 2
	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("a", 1, 1, now, future, 5),
		walInsertMaxRuns("a", 2, 1, now.Add(time.Second), future, 2),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	sq := c.getOrCreateSubQueue("a")
	if sq.maxRuns != 2 {
		t.Fatalf("maxRuns = %d, want 2 (newer task's value)", sq.maxRuns)
	}

	// the older task's slot is re-inserted (as a replay does) with its original
	// timestamp and stale value; the newer task's limit must survive
	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("a", 1, 1, now, future, 5),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	if sq.maxRuns != 2 {
		t.Fatalf("maxRuns = %d after old-task re-insert, want 2", sq.maxRuns)
	}
}

// UPDATE messages (retry resync of an existing slot) and DELETE messages never carry a
// value the index should apply: only INSERT observations may move the limit.
func TestDynamicMaxRuns_UpdateAndDeleteDoNotMoveLimit(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newGroupRoundRobinStrategy(repo, 1)

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("a", 1, 1, now, future, 4),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	stale := int32(9)
	updateMsg := walMessage{
		Operation:           "UPDATE",
		Key:                 "a",
		TaskId:              1,
		TaskInsertedAt:      now.Add(time.Minute), // even a newer timestamp must not apply on UPDATE
		ScheduleTimeoutAtMs: future.UnixMilli(),
		MaxRuns:             &stale,
	}
	deleteMsg := walMessage{
		Operation:      "DELETE",
		Key:            "a",
		TaskId:         1,
		TaskInsertedAt: now.Add(time.Minute),
		MaxRuns:        &stale,
	}

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{updateMsg, deleteMsg}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	sq := c.getOrCreateSubQueue("a")
	if sq.maxRuns != 4 {
		t.Fatalf("maxRuns = %d after UPDATE/DELETE, want 4", sq.maxRuns)
	}
}

// A raised limit arrives on a slot INSERT and that same batch runs the decide step, so
// the extra capacity fills immediately without waiting for another event.
func TestDynamicMaxRuns_RaiseFillsBacklogInSameBatch(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newGroupRoundRobinStrategy(repo, 1)

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("a", 1, 1, now, future, 1),
		walInsertMaxRuns("a", 2, 1, now.Add(time.Millisecond), future, 1),
		walInsertMaxRuns("a", 3, 1, now.Add(2*time.Millisecond), future, 1),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 1 || sq.queued.len() != 2 {
		t.Fatalf("before raise: (running %d, queued %d), want (1, 2)", sq.running.len(), sq.queued.len())
	}

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("a", 4, 1, now.Add(time.Second), future, 4),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	if sq.running.len() != 4 || sq.queued.len() != 0 {
		t.Fatalf("after raise: (running %d, queued %d), want (4, 0)", sq.running.len(), sq.queued.len())
	}
}

// GROUP_ROUND_ROBIN grandfathers running work when the limit drops: no new fills until
// the group drains below the new limit, but nothing running is cancelled.
func TestDynamicMaxRuns_LowerGrandfathersRunning_GroupRoundRobin(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newGroupRoundRobinStrategy(repo, 1)

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("a", 1, 1, now, future, 3),
		walInsertMaxRuns("a", 2, 1, now.Add(time.Millisecond), future, 3),
		walInsertMaxRuns("a", 3, 1, now.Add(2*time.Millisecond), future, 3),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("a", 4, 1, now.Add(time.Second), future, 1),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 3 {
		t.Fatalf("running = %d, want 3 (grandfathered above the lowered limit)", sq.running.len())
	}
	if _, ok := sq.queued.get(4); !ok {
		t.Fatalf("task 4 should stay queued until the group drains below the new limit")
	}
	if got := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit); len(got) != 0 {
		t.Fatalf("GROUP_ROUND_ROBIN must not cancel running work on a lowered limit: %v", got)
	}
}

// CANCEL_IN_PROGRESS actively trims running work above a lowered limit (it already trims
// running.len() > maxRuns for static limits; a dynamic drop takes the same path).
func TestDynamicMaxRuns_LowerTrimsRunning_CancelInProgress(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newTestStrategyKind(repo, 1, sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS)

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("a", 1, 1, now, future, 3),
		walInsertMaxRuns("a", 2, 1, now.Add(time.Millisecond), future, 3),
		walInsertMaxRuns("a", 3, 1, now.Add(2*time.Millisecond), future, 3),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 3 {
		t.Fatalf("running = %d before drop, want 3", sq.running.len())
	}

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("a", 4, 9, now.Add(time.Second), future, 1),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	if sq.running.len() != 1 {
		t.Fatalf("running = %d after drop to 1, want 1 (CANCEL_IN_PROGRESS trims)", sq.running.len())
	}
	if got := cancelledByReason(repo.lastCancelled, repository.CancelledReasonConcurrencyLimit); len(got) == 0 {
		t.Fatalf("expected CONCURRENCY_LIMIT cancellations when the limit dropped")
	}
}

// A failed flush must restore the pre-batch limit along with the index membership: the
// batch that raised the limit was never durable, so its observation must not survive.
func TestDynamicMaxRuns_RollbackRestoresLimit(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newGroupRoundRobinStrategy(repo, 1)

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("a", 1, 1, now, future, 2),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	repo.updateErr = errors.New("db unavailable")

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("a", 2, 1, now.Add(time.Second), future, 10),
	}); err == nil {
		t.Fatalf("expected error from failed flush")
	}
	c.rollbackScopes()

	sq := c.getOrCreateSubQueue("a")
	if sq.maxRuns != 2 {
		t.Fatalf("maxRuns = %d after rollback, want 2", sq.maxRuns)
	}
	if sq.maxRunsFrom != now.UnixNano() {
		t.Fatalf("maxRunsFrom not restored: got %d, want %d", sq.maxRunsFrom, now.UnixNano())
	}

	// the redelivered message must apply cleanly after the failure clears
	repo.updateErr = nil
	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("a", 2, 1, now.Add(time.Second), future, 10),
	}); err != nil {
		t.Fatalf("processWALMessages after retry: %v", err)
	}
	c.commitScopes()

	if sq.maxRuns != 10 {
		t.Fatalf("maxRuns = %d after redelivery, want 10", sq.maxRuns)
	}
}

// Hydration applies per-row observations under the same timestamp guard, so the group
// converges to the newest task's value regardless of row order.
func TestDynamicMaxRuns_HydrationConvergesToNewestTask(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	// newest-first order: the older row arrives second and must not win
	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRowMaxRuns("a", 2, 1, 0, now.Add(time.Second), future, false, 7),
			indexRowMaxRuns("a", 1, 1, 0, now, future, true, 3),
		},
	}
	c := newTestStrategy(repo, 1)

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.maxRuns != 7 {
		t.Fatalf("maxRuns = %d after hydration, want 7 (newest task's value)", sq.maxRuns)
	}

	// slots without a value (static strategies) leave the construction default intact
	repo2 := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("b", 3, 1, 0, now, future, false),
		},
	}
	c2 := newTestStrategy(repo2, 5)
	if err := c2.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}
	if sq2 := c2.getOrCreateSubQueue("b"); sq2.maxRuns != 5 {
		t.Fatalf("maxRuns = %d for static rows, want construction default 5", sq2.maxRuns)
	}
}

// Equal timestamps stay last-write-wins so same-instant inserts behave like any other
// insert order; older timestamps are rejected.
func TestObserveMaxRunsTimestampGuard(t *testing.T) {
	sq := newSubQueue("k", 1, priorityCompare)

	base := time.Now().UTC().UnixNano()

	sq.observeMaxRuns(5, base)
	if sq.maxRuns != 5 {
		t.Fatalf("maxRuns = %d, want 5", sq.maxRuns)
	}

	sq.observeMaxRuns(3, base) // equal timestamp: applies (last write wins)
	if sq.maxRuns != 3 {
		t.Fatalf("maxRuns = %d after equal-timestamp observation, want 3", sq.maxRuns)
	}

	sq.observeMaxRuns(9, base-1) // older: rejected
	if sq.maxRuns != 3 {
		t.Fatalf("maxRuns = %d after older observation, want 3", sq.maxRuns)
	}

	sq.observeMaxRuns(9, base+1) // newer: applies
	if sq.maxRuns != 9 {
		t.Fatalf("maxRuns = %d after newer observation, want 9", sq.maxRuns)
	}
}

// An in-place static-limit update must not clobber a dynamically observed group limit:
// only groups still on the static default follow the new value.
func TestDynamicMaxRuns_UpdateStrategyPreservesObservedLimits(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newGroupRoundRobinStrategy(repo, 1)

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsertMaxRuns("dynamic", 1, 1, now, future, 5),
		walInsert("static", 2, 1, now, future),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	c.UpdateStrategy(&sqlcv1.V1StepConcurrency{
		MaxConcurrency: 2,
		Strategy:       sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN,
	})

	if sq := c.getOrCreateSubQueue("dynamic"); sq.maxRuns != 5 {
		t.Fatalf("dynamic group maxRuns = %d after static update, want observed 5", sq.maxRuns)
	}
	if sq := c.getOrCreateSubQueue("static"); sq.maxRuns != 2 {
		t.Fatalf("static group maxRuns = %d after static update, want 2", sq.maxRuns)
	}
}
