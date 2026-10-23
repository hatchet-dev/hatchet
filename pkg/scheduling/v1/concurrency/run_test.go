package concurrency

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hatchet-dev/pgoutbox"
	outboxsqlc "github.com/hatchet-dev/pgoutbox/sqlc"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// fakeOutbox feeds scripted batches to the strategy's Flush, one batch per ProcessMessages call. A
// batch with commitErr fails after Flush returns, like a failed message delete or commit.
// afterFlush, if set, runs between Flush and the commit.
type fakeOutbox struct {
	flusher    pgoutbox.Flusher
	batches    []fakeBatch
	afterFlush func()
}

type fakeBatch struct {
	msgs      []walMessage
	commitErr error
}

type fakeFlushContext struct{ context.Context }

func (fakeFlushContext) Tx() pgx.Tx { return nil }

func (o *fakeOutbox) AddFlusher(topic string, flusher pgoutbox.Flusher) { o.flusher = flusher }

func (o *fakeOutbox) ProcessMessages(ctx context.Context, topic string, opts ...pgoutbox.ProcessOpt) ([]*outboxsqlc.Message, error) {
	if len(o.batches) == 0 {
		return nil, nil
	}

	b := o.batches[0]
	o.batches = o.batches[1:]

	msgs := make([]*outboxsqlc.Message, len(b.msgs))
	for i, m := range b.msgs {
		payload, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		msgs[i] = &outboxsqlc.Message{Payload: payload}
	}

	if err := o.flusher.Flush(fakeFlushContext{ctx}, msgs); err != nil {
		return nil, err
	}

	if o.afterFlush != nil {
		o.afterFlush()
	}

	if b.commitErr != nil {
		return nil, b.commitErr
	}

	return msgs, nil
}

func (o *fakeOutbox) AddMessages(ctx context.Context, tx pgx.Tx, topic string, msgs []pgoutbox.MessageOpts, opts ...pgoutbox.AddOpt) error {
	return nil
}

func (o *fakeOutbox) Subscribe(ctx context.Context, topic string, opts ...pgoutbox.SubscribeOpt) error {
	return nil
}

func (o *fakeOutbox) AcquireTopic(ctx context.Context, topic string) error { return nil }

func (o *fakeOutbox) ReleaseTopic(ctx context.Context, topic string) error { return nil }

func queuedIDs(res *repository.RunConcurrencyResult) []int64 {
	ids := make([]int64, len(res.Queued))
	for i, q := range res.Queued {
		ids[i] = q.Id
	}
	return ids
}

// When a batch fails to commit, Run must still return the results of the work that committed
// before it in the same Run (the initial queueing pass and earlier batches), and only those.
func TestRunReturnsCommittedResultsWhenALaterBatchFails(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	repo := &mockConcurrencyRepo{
		indexRows: []*sqlcv1.ListConcurrencySlotsForIndexingRow{
			indexRow("a", 1, 5, 0, now, future, false),
		},
	}
	outbox := &fakeOutbox{
		batches: []fakeBatch{
			{msgs: []walMessage{walInsert("a", 2, 5, now, future)}},
			{msgs: []walMessage{walInsert("a", 3, 5, now, future)}, commitErr: errors.New("commit failed")},
		},
	}

	c := newTestStrategy(repo, 5)
	c.outbox = outbox
	outbox.AddFlusher(c.topic, c)

	if err := c.buildIndex(context.Background()); err != nil {
		t.Fatalf("buildIndex: %v", err)
	}
	close(c.built)

	res, err := c.Run(context.Background())
	if err == nil {
		t.Fatalf("expected an error from the failed batch")
	}
	if res == nil {
		t.Fatalf("committed results were dropped")
	}

	got := queuedIDs(res)
	if len(got) != 2 || !containsID(got, 1) || !containsID(got, 2) {
		t.Fatalf("queued = %v, want the initial pass (1) and the committed batch (2) only", got)
	}

	if _, ok := c.getOrCreateSubQueue("a").running.get(3); ok {
		t.Fatalf("the failed batch's slot was not rolled back")
	}
}

// UpdateStrategy can run on another goroutine while a batch's scopes are still open, between Flush
// and Run committing or rolling them back. Run with -race.
func TestUpdateStrategyDuringOpenBatch(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	for _, commitErr := range []error{nil, errors.New("commit failed")} {
		var wg sync.WaitGroup

		outbox := &fakeOutbox{
			batches: []fakeBatch{{msgs: []walMessage{walInsert("a", 1, 5, now, future)}, commitErr: commitErr}},
		}

		c := newTestStrategy(&mockConcurrencyRepo{}, 1)
		c.outbox = outbox
		outbox.AddFlusher(c.topic, c)
		close(c.built)

		outbox.afterFlush = func() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				c.UpdateStrategy(&sqlcv1.V1StepConcurrency{MaxConcurrency: 2, Strategy: c.strategy.Strategy})
			}()
		}

		_, _ = c.Run(context.Background())
		wg.Wait()

		if got := c.getOrCreateSubQueue("a").maxRuns; got != 2 {
			t.Fatalf("commitErr %v: maxRuns = %d after the update, want 2", commitErr, got)
		}
	}
}
