package concurrency

import (
	"context"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// UpdateStrategy must widen existing sub-queues in place — the queued backlog fills to
// the new limit on the next batch without a rebuild — and must not disturb index
// membership.
func TestUpdateStrategy_RaisesLimitInPlace(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newGroupRoundRobinStrategy(repo, 1)

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsert("a", 1, 1, now, future),
		walInsert("a", 2, 1, now.Add(time.Millisecond), future),
		walInsert("a", 3, 1, now.Add(2*time.Millisecond), future),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 1 || sq.queued.len() != 2 {
		t.Fatalf("before update: (running %d, queued %d), want (1, 2)", sq.running.len(), sq.queued.len())
	}

	next := &sqlcv1.V1StepConcurrency{
		MaxConcurrency: 3,
		Strategy:       sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN,
	}
	c.UpdateStrategy(next)

	if sq.maxRuns != 3 {
		t.Fatalf("maxRuns = %d after update, want 3", sq.maxRuns)
	}
	if sq.running.len() != 1 || sq.queued.len() != 2 {
		t.Fatalf("membership disturbed by update: (running %d, queued %d)", sq.running.len(), sq.queued.len())
	}

	// the next batch (here: an empty one for the sub-queue) fills the backlog to the new limit
	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsert("a", 4, 1, now.Add(time.Second), future),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	if sq.running.len() != 3 {
		t.Fatalf("running = %d after update + batch, want 3", sq.running.len())
	}

	// new sub-queues pick up the updated static limit too
	if sqB := c.getOrCreateSubQueue("b"); sqB.maxRuns != 3 {
		t.Fatalf("new sub-queue maxRuns = %d, want 3", sqB.maxRuns)
	}
}

// Lowering the static limit in place is grandfathered by GROUP_ROUND_ROBIN exactly like a
// rebuild would behave: running work is untouched, no new fills until the group drains.
func TestUpdateStrategy_LowersLimitInPlace(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newGroupRoundRobinStrategy(repo, 2)

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsert("a", 1, 1, now, future),
		walInsert("a", 2, 1, now.Add(time.Millisecond), future),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	c.UpdateStrategy(&sqlcv1.V1StepConcurrency{
		MaxConcurrency: 1,
		Strategy:       sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN,
	})

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsert("a", 3, 1, now.Add(time.Second), future),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 2 {
		t.Fatalf("running = %d, want 2 (grandfathered above the lowered limit)", sq.running.len())
	}
	if _, ok := sq.queued.get(3); !ok {
		t.Fatalf("task 3 should stay queued under the lowered limit")
	}
}

// A raised limit must reach groups that see no further WAL traffic: UpdateStrategy
// re-arms the all-sub-queue queueing pass, so the next pass promotes idle backlog
// without any new message touching the group.
func TestUpdateStrategy_RequeuesIdleBacklog(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{}
	c := newGroupRoundRobinStrategy(repo, 1)

	if _, err := c.processWALMessages(context.Background(), nil, []walMessage{
		walInsert("a", 1, 1, now, future),
		walInsert("a", 2, 1, now.Add(time.Millisecond), future),
		walInsert("a", 3, 1, now.Add(2*time.Millisecond), future),
	}); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.commitScopes()

	// normal operation state: the post-build pass has already run
	c.initialQueued = true

	c.UpdateStrategy(&sqlcv1.V1StepConcurrency{
		MaxConcurrency: 3,
		Strategy:       sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN,
	})

	if c.initialQueued {
		t.Fatalf("UpdateStrategy must re-arm the all-sub-queue queueing pass")
	}

	// the pass the next Run performs — no WAL message for group "a" anywhere
	if _, err := c.runInitialQueueing(context.Background()); err != nil {
		t.Fatalf("runInitialQueueing: %v", err)
	}

	sq := c.getOrCreateSubQueue("a")
	if sq.running.len() != 3 || sq.queued.len() != 0 {
		t.Fatalf("idle backlog not promoted: (running %d, queued %d), want (3, 0)", sq.running.len(), sq.queued.len())
	}
}

// Lowering the limit below the current running count and immediately re-deciding every
// sub-queue (which the re-armed pass now does) must be safe for every strategy kind:
// no panics, and running work is trimmed or grandfathered per the kind's semantics.
func TestUpdateStrategy_LowerThenRequeueAllKinds(t *testing.T) {
	kinds := []sqlcv1.V1ConcurrencyStrategy{
		sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN,
		sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS,
		sqlcv1.V1ConcurrencyStrategyCANCELNEWEST,
		sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTNEWEST,
		sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTOLDEST,
	}

	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			now := time.Now().UTC()
			future := now.Add(time.Hour)

			repo := &mockConcurrencyRepo{}
			c := newTestStrategyKind(repo, 3, kind)

			// 3 running + 2 queued
			msgs := make([]walMessage, 0, 5)
			for i := int64(1); i <= 5; i++ {
				msgs = append(msgs, walInsert("a", i, 1, now.Add(time.Duration(i)*time.Millisecond), future))
			}
			if _, err := c.processWALMessages(context.Background(), nil, msgs); err != nil {
				t.Fatalf("processWALMessages: %v", err)
			}
			c.commitScopes()

			sq := c.getOrCreateSubQueue("a")
			running := sq.running.len()

			c.initialQueued = true
			c.UpdateStrategy(&sqlcv1.V1StepConcurrency{MaxConcurrency: 1, Strategy: kind})

			if _, err := c.runInitialQueueing(context.Background()); err != nil {
				t.Fatalf("runInitialQueueing after lower: %v", err)
			}
			c.commitScopes()

			switch kind {
			case sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS:
				if sq.running.len() != 1 {
					t.Fatalf("running = %d, want trimmed to 1", sq.running.len())
				}
			default:
				if sq.running.len() != running {
					t.Fatalf("running = %d, want grandfathered %d", sq.running.len(), running)
				}
			}
		})
	}
}
