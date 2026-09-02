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
