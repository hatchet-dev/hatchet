package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// The resident counts behind the index gauges must match the sub-queues after a build, the
// post-build pass, committed batches and rolled back batches.
func TestResidentCountsMatchSubQueues(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)
	ctx := context.Background()

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 0, 0, now, future, true),
			indexRow("a", 2, 0, 0, now.Add(time.Second), future, false),
			indexRow("a", 3, 0, 0, now.Add(2*time.Second), future, false),
			indexRow("b", 4, 0, 0, now, future, false),
		},
	}
	c := newTestStrategyKind(repo, 1, sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTNEWEST)

	check := func(step string) {
		t.Helper()
		if got, want := c.resident.load(), c.countResident(); got != want {
			t.Fatalf("%s: resident = %+v, want %+v", step, got, want)
		}
	}

	if err := c.buildIndex(ctx); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}
	check("build")

	if _, err := c.queueAllSubQueues(ctx); err != nil {
		t.Fatalf("queueAllSubQueues: %v", err)
	}
	check("initial pass")

	batch := []walMessage{
		walInsert("a", 5, 0, now.Add(3*time.Second), future),
		walInsert("c", 6, 0, now, future),
		{Operation: "DELETE", Key: "b", TaskId: 4},
	}
	if _, err := c.processWALMessages(ctx, nil, batch); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.pruneEmpty(c.commitScopes())
	check("committed batch")

	repo.updateErr = errors.New("db unavailable")
	failed := []walMessage{
		{Operation: "DELETE", Key: "a", TaskId: 1},
		walInsert("d", 7, 0, now, future),
	}
	if _, err := c.processWALMessages(ctx, nil, failed); err == nil {
		t.Fatalf("expected the flush to fail")
	}
	c.rollbackScopes()
	check("rolled back batch")

	repo.updateErr = nil
	if _, err := c.processWALMessages(ctx, nil, failed); err != nil {
		t.Fatalf("processWALMessages: %v", err)
	}
	c.pruneEmpty(c.commitScopes())
	check("retried batch")
}
