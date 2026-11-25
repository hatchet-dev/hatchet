//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/pgoutbox"
	outboxsqlc "github.com/hatchet-dev/pgoutbox/sqlc"
	"github.com/jackc/pgx/v5"
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
