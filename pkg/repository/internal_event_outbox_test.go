//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/pgoutbox"
	outboxsqlc "github.com/hatchet-dev/pgoutbox/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeOutbox flushes a one-message batch per ProcessMessages call through the registered flusher,
// so every pass runs whichever handler the outbox picks.
type fakeOutbox struct {
	pgoutbox.Outbox

	mu       sync.Mutex
	flushers map[string]pgoutbox.Flusher
	calls    atomic.Int64
	added    map[string][][]byte
}

func (f *fakeOutbox) AddMessages(_ context.Context, _ pgx.Tx, topic string, msgs []pgoutbox.MessageOpts, _ ...pgoutbox.AddOpt) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.added == nil {
		f.added = map[string][][]byte{}
	}

	for _, m := range msgs {
		f.added[topic] = append(f.added[topic], m.Payload)
	}

	return nil
}

type fakeFlushContext struct {
	context.Context
}

func (fakeFlushContext) Tx() pgx.Tx { return nil }

func (f *fakeOutbox) AddFlusher(topic string, flusher pgoutbox.Flusher) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.flushers[topic] = flusher
}

func (f *fakeOutbox) ProcessMessages(ctx context.Context, topic string, _ ...pgoutbox.ProcessOpt) ([]*outboxsqlc.Message, error) {
	f.calls.Add(1)

	f.mu.Lock()
	flusher := f.flushers[topic]
	f.mu.Unlock()

	payload, err := json.Marshal(internalEventOutboxMessage{TenantID: uuid.New()})

	if err != nil {
		return nil, err
	}

	// an empty result ends the pass, so the loop waits for its next poll instead of spinning
	return nil, flusher.Flush(fakeFlushContext{ctx}, []*outboxsqlc.Message{{Payload: payload}})
}

func newTestInternalEventOutbox(writes bool) (*InternalEventOutbox, *fakeOutbox) {
	fake := &fakeOutbox{flushers: map[string]pgoutbox.Flusher{}}
	l := zerolog.Nop()

	o := NewInternalEventOutbox(fake, writes, 5*time.Millisecond, &l)
	o.idleInterval = 5 * time.Millisecond

	return o, fake
}

type countingHandler struct {
	calls atomic.Int64
}

func (h *countingHandler) handle(context.Context, uuid.UUID, []*InternalTaskEvent) error {
	h.calls.Add(1)
	return nil
}

func waitForCalls(t *testing.T, calls *atomic.Int64, after int64) {
	t.Helper()

	require.Eventually(t, func() bool { return calls.Load() > after }, 2*time.Second, time.Millisecond)
}

func TestInternalEventOutboxDrainsUntilTheLastHandlerStops(t *testing.T) {
	for _, writes := range []bool{true, false} {
		o, fake := newTestInternalEventOutbox(writes)
		controller, direct := &countingHandler{}, &countingHandler{}

		stopController := o.Start(controller.handle)
		stopDirect := o.Start(direct.handle)
		waitForCalls(t, &controller.calls, 0)

		// in a process with both, either can stop first and the other's handler takes over
		stopController()
		stopController()
		waitForCalls(t, &direct.calls, 0)

		stopDirect()
		after := fake.calls.Load()
		time.Sleep(30 * time.Millisecond)
		assert.Equal(t, after, fake.calls.Load(), "writes=%t: a loop kept draining after every handler stopped", writes)
	}
}

// blockingHandler holds every batch until released, or until its context is cancelled.
type blockingHandler struct {
	running  chan struct{}
	release  chan struct{}
	finished atomic.Bool
	aborted  atomic.Bool
}

func newBlockingHandler() *blockingHandler {
	return &blockingHandler{running: make(chan struct{}, 100), release: make(chan struct{})}
}

func (h *blockingHandler) handle(ctx context.Context, _ uuid.UUID, _ []*InternalTaskEvent) error {
	h.running <- struct{}{}

	select {
	case <-h.release:
		h.finished.Store(true)
		return nil
	case <-ctx.Done():
		h.aborted.Store(true)
		return ctx.Err()
	}
}

func stopAsync(stop func()) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		stop()
		close(done)
	}()

	return done
}

// A non-last stop returns only once no batch still runs the stopping handler, since its owner then
// tears down what the handler uses.
func TestInternalEventOutboxNonLastStopWaitsForItsBatch(t *testing.T) {
	o, _ := newTestInternalEventOutbox(true)
	first := newBlockingHandler()

	stopFirst := o.Start(first.handle)
	<-first.running

	stopSecond := o.Start((&countingHandler{}).handle)
	defer stopSecond()

	done := stopAsync(stopFirst)

	select {
	case <-done:
		t.Fatal("stop returned while a batch still ran the handler")
	case <-time.After(30 * time.Millisecond):
	}

	close(first.release)
	<-done
	assert.True(t, first.finished.Load())
}

// The last stop lets a running batch finish, since a handler cancelled after committing its matches
// can't redo the signalling that follows.
func TestInternalEventOutboxLastStopLetsTheBatchFinish(t *testing.T) {
	o, _ := newTestInternalEventOutbox(true)
	h := newBlockingHandler()

	stop := o.Start(h.handle)
	<-h.running

	done := stopAsync(stop)

	select {
	case <-done:
		t.Fatal("stop returned while a batch still ran")
	case <-time.After(30 * time.Millisecond):
	}

	close(h.release)
	<-done
	assert.True(t, h.finished.Load())
	assert.False(t, h.aborted.Load())
}

func TestInternalEventOutboxLastStopCancelsABatchPastTheTimeout(t *testing.T) {
	o, _ := newTestInternalEventOutbox(true)
	o.stopTimeout = 20 * time.Millisecond
	h := newBlockingHandler()

	stop := o.Start(h.handle)
	<-h.running

	select {
	case <-stopAsync(stop):
	case <-time.After(2 * time.Second):
		t.Fatal("stop didn't cancel a batch past its timeout")
	}

	assert.True(t, h.aborted.Load())
}

// Close is the backstop for handlers their owner never stopped: it stops the loops regardless, and
// a later Start doesn't bring them back on a closed pool.
func TestInternalEventOutboxCloseStopsLeakedHandlers(t *testing.T) {
	o, fake := newTestInternalEventOutbox(true)
	h := &countingHandler{}

	o.Start(h.handle)
	waitForCalls(t, &h.calls, 0)

	o.Close()
	o.Start(h.handle)

	after := fake.calls.Load()
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, after, fake.calls.Load())
}

func TestInternalEventOutboxStartRacingTheLastStop(t *testing.T) {
	o, _ := newTestInternalEventOutbox(true)

	var wg sync.WaitGroup

	for range 50 {
		stop := o.Start((&countingHandler{}).handle)

		wg.Add(2)

		go func() {
			defer wg.Done()
			stop()
		}()

		go func() {
			defer wg.Done()
			o.Start((&countingHandler{}).handle)()
		}()
	}

	wg.Wait()
}

func outboxMessage(t *testing.T, id int64, topic string, m internalEventOutboxMessage) *outboxsqlc.Message {
	t.Helper()

	payload, err := json.Marshal(m)
	require.NoError(t, err)

	return &outboxsqlc.Message{ID: id, Topic: topic, Payload: payload, InsertedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}}
}

func retried(t *testing.T, fake *fakeOutbox) []internalEventOutboxMessage {
	t.Helper()

	res := make([]internalEventOutboxMessage, 0)

	for _, payload := range fake.added[InternalEventRetryTopic(0)] {
		var m internalEventOutboxMessage
		require.NoError(t, json.Unmarshal(payload, &m))
		res = append(res, m)
	}

	return res
}

// flusherWith registers handler and returns the flusher for topic, without loops racing the test.
func flusherWith(t *testing.T, o *InternalEventOutbox, fake *fakeOutbox, topic string, handler InternalEventHandler) pgoutbox.Flusher {
	t.Helper()

	o.Start(handler)()
	o.handlers = []*registeredHandler{{handler: handler}}

	return fake.flushers[topic]
}

// One tenant's failing event must not hold up anything else: the rest of the batch commits, the
// tenant's other messages are run on their own, and only the failing one moves to the retry topic.
func TestInternalEventOutboxRetriesOnlyTheFailingMessage(t *testing.T) {
	o, fake := newTestInternalEventOutbox(true)
	tenant, other := uuid.New(), uuid.New()
	poison := InternalTaskEvent{EventKey: "poison"}
	var handledAlone []string

	flusher := flusherWith(t, o, fake, InternalEventOutboxTopic(0), func(_ context.Context, tenantId uuid.UUID, events []*InternalTaskEvent) error {
		for _, e := range events {
			if e.EventKey == "poison" {
				return errors.New("boom")
			}
		}

		if tenantId == tenant && len(events) == 1 {
			handledAlone = append(handledAlone, events[0].EventKey)
		}

		return nil
	})

	firstAt := time.Now().Add(-time.Hour)
	err := flusher.Flush(fakeFlushContext{context.Background()}, []*outboxsqlc.Message{
		outboxMessage(t, 1, InternalEventOutboxTopic(0), internalEventOutboxMessage{TenantID: tenant, Events: []InternalTaskEvent{poison}, Attempts: 2, FirstAt: firstAt}),
		outboxMessage(t, 2, InternalEventOutboxTopic(0), internalEventOutboxMessage{TenantID: tenant, Events: []InternalTaskEvent{{EventKey: "healthy"}}}),
		outboxMessage(t, 3, InternalEventOutboxTopic(0), internalEventOutboxMessage{TenantID: other, Events: []InternalTaskEvent{{EventKey: "other"}}}),
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"healthy"}, handledAlone)
	assert.Empty(t, fake.added[InternalEventOutboxTopic(0)], "nothing goes back on the live topic")

	r := retried(t, fake)
	require.Len(t, r, 1)
	assert.Equal(t, "poison", r[0].Events[0].EventKey)
	assert.Equal(t, 3, r[0].Attempts)
	assert.True(t, r[0].RetryAt.After(time.Now()))
	assert.True(t, firstAt.Equal(r[0].FirstAt), "a retry keeps the original write time")
}

func TestInternalEventOutboxHoldsBackARetryUntilItIsDue(t *testing.T) {
	o, fake := newTestInternalEventOutbox(true)
	h := &countingHandler{}
	flusher := flusherWith(t, o, fake, InternalEventRetryTopic(0), h.handle)
	waiting := internalEventOutboxMessage{TenantID: uuid.New(), Attempts: 4, RetryAt: time.Now().Add(time.Minute)}

	require.NoError(t, flusher.Flush(fakeFlushContext{context.Background()}, []*outboxsqlc.Message{outboxMessage(t, 1, InternalEventRetryTopic(0), waiting)}))

	assert.Zero(t, h.calls.Load())

	r := retried(t, fake)
	require.Len(t, r, 1)
	assert.Equal(t, 4, r[0].Attempts, "a retry held back isn't an attempt")
}

// A cancelled batch is shutdown, not a failure: it rolls back whole and nothing moves to retries.
func TestInternalEventOutboxCancelledBatchIsNotRetried(t *testing.T) {
	o, fake := newTestInternalEventOutbox(true)
	flusher := flusherWith(t, o, fake, InternalEventOutboxTopic(0), func(ctx context.Context, _ uuid.UUID, _ []*InternalTaskEvent) error { return ctx.Err() })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := flusher.Flush(fakeFlushContext{ctx}, []*outboxsqlc.Message{outboxMessage(t, 1, InternalEventOutboxTopic(0), internalEventOutboxMessage{TenantID: uuid.New()})})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, fake.added)
}

func TestInternalEventOutboxMovesUndecodableMessagesToRetries(t *testing.T) {
	o, fake := newTestInternalEventOutbox(true)
	flusher := flusherWith(t, o, fake, InternalEventOutboxTopic(3), (&countingHandler{}).handle)

	require.NoError(t, flusher.Flush(fakeFlushContext{context.Background()}, []*outboxsqlc.Message{{ID: 1, Topic: InternalEventOutboxTopic(3), Payload: []byte("not json")}}))

	assert.Equal(t, [][]byte{[]byte("not json")}, fake.added[InternalEventRetryTopic(3)])
}

func TestInternalEventRetryDelayIsCapped(t *testing.T) {
	assert.Equal(t, 2*time.Second, internalEventRetryDelay(1))
	assert.Equal(t, 32*time.Second, internalEventRetryDelay(5))
	assert.Equal(t, internalEventRetryCap, internalEventRetryDelay(6))
	assert.Equal(t, internalEventRetryCap, internalEventRetryDelay(1000))
}

// Measured against real pgoutbox, which only ever takes a topic's oldest messages: hundreds of events
// that keep failing must not delay a good event behind them on the same shard.
func TestInternalEventOutboxFailingEventsDontDelayTheirShard(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(func() { cleanup() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	ob, err := pgoutbox.NewOutbox(ctx, pool, pgoutbox.WithAutoMigrate(false))
	require.NoError(t, err)

	l := zerolog.Nop()
	o := NewInternalEventOutbox(ob, true, time.Second, &l)

	bad := uuid.New()
	good := uuid.New()

	for o.shardFor(good) != o.shardFor(bad) {
		good = uuid.New()
	}

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)

	for range 1000 {
		require.NoError(t, o.enqueue(ctx, tx, bad, []InternalTaskEvent{{EventKey: "poison"}}))
	}

	require.NoError(t, tx.Commit(ctx))

	handledGood := make(chan struct{}, 1)

	stop := o.Start(func(_ context.Context, tenantId uuid.UUID, _ []*InternalTaskEvent) error {
		if tenantId == bad {
			return errors.New("boom")
		}

		select {
		case handledGood <- struct{}{}:
		default:
		}

		return nil
	})
	defer stop()

	// wait for the first poll to fail the poison events once and move every one of them to retries
	countMessages := func(topic string) int {
		var n int
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM outbox.messages WHERE topic = $1`, topic).Scan(&n))
		return n
	}

	shard := o.shardFor(bad)
	require.Eventually(t, func() bool {
		return countMessages(InternalEventOutboxTopic(shard)) == 0 && countMessages(InternalEventRetryTopic(shard)) == 1000
	}, 30*time.Second, 50*time.Millisecond)

	tx, err = pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, o.enqueue(ctx, tx, good, []InternalTaskEvent{{EventKey: "good"}}))
	require.NoError(t, tx.Commit(ctx))

	written := time.Now()
	o.Kick(good)

	select {
	case <-handledGood:
		assert.Less(t, time.Since(written), 500*time.Millisecond)
	case <-time.After(5 * time.Second):
		t.Fatal("a good event waited behind failing ones on its shard")
	}
}

// A handler that hangs holds its topic's lock for every engine, so a call is bounded and then retried.
func TestInternalEventOutboxRetriesAHangingHandler(t *testing.T) {
	o, fake := newTestInternalEventOutbox(true)
	o.handlerTimeout = 20 * time.Millisecond

	flusher := flusherWith(t, o, fake, InternalEventOutboxTopic(0), func(ctx context.Context, _ uuid.UUID, _ []*InternalTaskEvent) error {
		<-ctx.Done()
		return ctx.Err()
	})

	require.NoError(t, flusher.Flush(fakeFlushContext{context.Background()}, []*outboxsqlc.Message{outboxMessage(t, 1, InternalEventOutboxTopic(0), internalEventOutboxMessage{TenantID: uuid.New()})}))

	assert.Len(t, retried(t, fake), 1)
}

// A tenant-wide failure would otherwise cost one call per message under the topic lock.
func TestInternalEventOutboxCapsOneByOneReruns(t *testing.T) {
	o, fake := newTestInternalEventOutbox(true)
	tenant := uuid.New()
	var calls atomic.Int64

	flusher := flusherWith(t, o, fake, InternalEventOutboxTopic(0), func(context.Context, uuid.UUID, []*InternalTaskEvent) error {
		calls.Add(1)
		return errors.New("tenant is broken")
	})

	msgs := make([]*outboxsqlc.Message, 0, 10)

	for i := range 10 {
		msgs = append(msgs, outboxMessage(t, int64(i), InternalEventOutboxTopic(0), internalEventOutboxMessage{TenantID: tenant}))
	}

	require.NoError(t, flusher.Flush(fakeFlushContext{context.Background()}, msgs))

	assert.Equal(t, int64(1+maxIndividualReruns), calls.Load())
	assert.Len(t, retried(t, fake), 10)
}
