package directwrite

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

type sendCall struct {
	ctxErr      error
	hasDeadline bool
}

type fakeMQ struct {
	mu    sync.Mutex
	sent  []*msgqueue.Message
	calls []sendCall
	err   error

	// failures makes the first n sends fail, to exercise the fallback's retries
	failures int
}

func (m *fakeMQ) Clone() (func() error, msgqueue.MessageQueue, error) {
	return func() error { return nil }, m, nil
}
func (m *fakeMQ) SetQOS(int) {}
func (m *fakeMQ) SendMessage(ctx context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if q.Name() != msgqueue.TASK_PROCESSING_QUEUE.Name() {
		return errors.New("unexpected queue")
	}

	_, hasDeadline := ctx.Deadline()
	m.calls = append(m.calls, sendCall{ctxErr: ctx.Err(), hasDeadline: hasDeadline})

	if m.failures > 0 {
		m.failures--
		return errors.New("publish failed")
	}

	m.sent = append(m.sent, msg)
	return m.err
}

func (m *fakeMQ) callsSnapshot() []sendCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]sendCall(nil), m.calls...)
}
func (m *fakeMQ) Subscribe(msgqueue.Queue, msgqueue.MsgHandler, msgqueue.MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}
func (m *fakeMQ) IsReady() bool { return true }

func (m *fakeMQ) sentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sent)
}

func newMsg(t *testing.T) *msgqueue.Message {
	t.Helper()

	msg, err := msgqueue.NewTenantMessage(uuid.New(), msgqueue.MsgIDTaskCompleted, false, true, map[string]string{"k": "v"})
	require.NoError(t, err)

	return msg
}

func newWriter(t *testing.T, dst msgqueue.DstFunc, mq msgqueue.MessageQueue, opts Opts) *Writer {
	t.Helper()

	l := zerolog.Nop()
	w, err := New(dst, mq, &l, opts)
	require.NoError(t, err)

	return w
}

func TestWriteAppliesInProcess(t *testing.T) {
	mq := &fakeMQ{}
	var handled atomic.Int32

	w := newWriter(t, func(_ uuid.UUID, msgId string, payloads [][]byte) error {
		assert.Equal(t, msgqueue.MsgIDTaskCompleted, msgId)
		handled.Add(int32(len(payloads)))
		return nil
	}, mq, Opts{Slots: 10})

	require.NoError(t, w.Write(context.Background(), newMsg(t)))
	assert.Equal(t, int32(1), handled.Load())
	assert.Equal(t, 0, mq.sentCount())
}

func TestWriteFallsBackOnHandlerError(t *testing.T) {
	mq := &fakeMQ{}

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		return errors.New("deadlock detected")
	}, mq, Opts{Slots: 10})

	require.NoError(t, w.Write(context.Background(), newMsg(t)))
	assert.Equal(t, 1, mq.sentCount())
}

func TestWriteFallsBackWhenSlotsAreTaken(t *testing.T) {
	mq := &fakeMQ{}
	release := make(chan struct{})
	started := make(chan struct{})

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		close(started)
		<-release
		return nil
	}, mq, Opts{Slots: 1})

	done := make(chan error)
	go func() { done <- w.Write(context.Background(), newMsg(t)) }()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first write was never handled")
	}

	require.NoError(t, w.Write(context.Background(), newMsg(t)))
	assert.Equal(t, 1, mq.sentCount())

	close(release)
	require.NoError(t, <-done)
	assert.Equal(t, 1, mq.sentCount())
}

func TestTryWriteAsyncFallsBackOnHandlerError(t *testing.T) {
	mq := &fakeMQ{}

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		return errors.New("deadlock detected")
	}, mq, Opts{Slots: 10})

	require.True(t, w.TryWriteAsync(context.Background(), newMsg(t)))

	require.Eventually(t, func() bool { return mq.sentCount() == 1 }, 5*time.Second, 10*time.Millisecond)
}

// Without a free slot TryWriteAsync leaves the message to the caller's batching publish path, and
// doesn't publish it itself or start a goroutine for it.
func TestTryWriteAsyncLeavesOverflowToTheCaller(t *testing.T) {
	mq := &fakeMQ{}
	release := make(chan struct{})

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		<-release
		return nil
	}, mq, Opts{Slots: 2})

	require.True(t, w.TryWriteAsync(context.Background(), newMsg(t)))
	require.True(t, w.TryWriteAsync(context.Background(), newMsg(t)))
	assert.False(t, w.TryWriteAsync(context.Background(), newMsg(t)))
	assert.Equal(t, 0, mq.sentCount())

	close(release)
	require.NoError(t, w.Close(context.Background()))
	assert.Equal(t, 0, mq.sentCount())
}

func TestCloseWaitsForBackgroundWrites(t *testing.T) {
	mq := &fakeMQ{}
	release := make(chan struct{})
	started := make(chan struct{})

	var handled, drainedAfterHandled atomic.Bool

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		close(started)
		<-release
		handled.Store(true)
		return nil
	}, mq, Opts{Slots: 10, OnDrained: func() { drainedAfterHandled.Store(handled.Load()) }})

	require.True(t, w.TryWriteAsync(context.Background(), newMsg(t)))
	<-started

	closed := make(chan error)
	go func() { closed <- w.Close(context.Background()) }()

	select {
	case <-closed:
		t.Fatal("Close returned while a write was still in flight")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	require.NoError(t, <-closed)
	assert.True(t, drainedAfterHandled.Load(), "OnDrained must run after the in-flight write finished")
}

func TestCloseGivesUpAtItsDeadline(t *testing.T) {
	mq := &fakeMQ{}
	release := make(chan struct{})
	started := make(chan struct{})

	var drained atomic.Bool

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		close(started)
		<-release
		return nil
	}, mq, Opts{Slots: 10, OnDrained: func() { drained.Store(true) }})

	require.True(t, w.TryWriteAsync(context.Background(), newMsg(t)))
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	require.ErrorIs(t, w.Close(ctx), context.DeadlineExceeded)
	assert.False(t, drained.Load(), "the handler's resources stay up while a write still uses them")

	close(release)
}

func TestWritesAfterCloseGoOverTheQueue(t *testing.T) {
	mq := &fakeMQ{}

	var applied atomic.Bool

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		applied.Store(true)
		return nil
	}, mq, Opts{Slots: 10})

	expired, cancel := context.WithCancel(context.Background())
	cancel()

	// nothing is in flight, so even an expired ctx lets Close finish
	require.NoError(t, w.Close(expired))
	require.NoError(t, w.Close(context.Background()), "a second Close must not panic")

	require.NoError(t, w.Write(context.Background(), newMsg(t)))
	assert.False(t, w.TryWriteAsync(context.Background(), newMsg(t)))
	assert.False(t, applied.Load())
	assert.Equal(t, 1, mq.sentCount())
}

func TestWriteReturnsQueueErrorOnFallback(t *testing.T) {
	setFastBackoff(t)

	mq := &fakeMQ{err: errors.New("broker down")}

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		return errors.New("deadlock detected")
	}, mq, Opts{Slots: 10})

	require.Error(t, w.Write(context.Background(), newMsg(t)))
}

func TestWriteFallsBackEvenWhenTheCallerIsGone(t *testing.T) {
	mq := &fakeMQ{}
	ctx, cancel := context.WithCancel(context.Background())

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		// the worker's RPC is cancelled while the handler runs, after it has committed part of the message
		cancel()
		return errors.New("could not send internal events")
	}, mq, Opts{Slots: 10})

	require.NoError(t, w.Write(ctx, newMsg(t)))

	calls := mq.callsSnapshot()
	require.Len(t, calls, 1)
	assert.NoError(t, calls[0].ctxErr, "the fallback publish must not inherit the cancelled RPC context")
	assert.True(t, calls[0].hasDeadline, "the detached publish still needs its own timeout")
}

func TestNoSlotFallbackIgnoresACancelledCaller(t *testing.T) {
	mq := &fakeMQ{}
	release := make(chan struct{})
	started := make(chan struct{})

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		close(started)
		<-release
		return nil
	}, mq, Opts{Slots: 1})

	first := make(chan error)
	go func() { first <- w.Write(context.Background(), newMsg(t)) }()
	<-started

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, w.Write(ctx, newMsg(t)))

	calls := mq.callsSnapshot()
	require.Len(t, calls, 1)
	assert.NoError(t, calls[0].ctxErr)
	assert.True(t, calls[0].hasDeadline)

	close(release)
	require.NoError(t, <-first)
}

func TestFallbackRetriesAFailedPublish(t *testing.T) {
	setFastBackoff(t)

	mq := &fakeMQ{failures: 2}

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		return errors.New("could not send internal events")
	}, mq, Opts{Slots: 10})

	require.NoError(t, w.Write(context.Background(), newMsg(t)))
	assert.Len(t, mq.callsSnapshot(), 3)
	assert.Equal(t, 1, mq.sentCount())
}

func TestFallbackGivesUpAfterItsAttempts(t *testing.T) {
	setFastBackoff(t)

	mq := &fakeMQ{failures: 10}

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		return errors.New("could not send internal events")
	}, mq, Opts{Slots: 10})

	require.Error(t, w.Write(context.Background(), newMsg(t)))
	assert.Len(t, mq.callsSnapshot(), fallbackAttempts)
}

func TestZeroSlotsSendsEverythingOverTheQueue(t *testing.T) {
	mq := &fakeMQ{}
	var applied atomic.Bool

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		applied.Store(true)
		return nil
	}, mq, Opts{Slots: 0})

	require.NoError(t, w.Write(context.Background(), newMsg(t)))
	assert.False(t, applied.Load())
	assert.Equal(t, 1, mq.sentCount())
}

func TestNegativeSlotsRemovesTheCap(t *testing.T) {
	mq := &fakeMQ{}
	release := make(chan struct{})

	var handled atomic.Int32

	w := newWriter(t, func(_ uuid.UUID, _ string, payloads [][]byte) error {
		handled.Add(int32(len(payloads))) // nolint: gosec
		<-release
		return nil
	}, mq, Opts{Slots: -1})

	// More concurrent writes than any small cap would admit, all held in the handler at once.
	const writes = 25

	var wg sync.WaitGroup

	for range writes {
		wg.Go(func() { assert.NoError(t, w.Write(context.Background(), newMsg(t))) })
	}

	require.Eventually(t, func() bool { return handled.Load() == writes }, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, 0, mq.sentCount())

	close(release)
	wg.Wait()
}

func TestCloseWaitsForSynchronousWrites(t *testing.T) {
	mq := &fakeMQ{}
	release := make(chan struct{})
	started := make(chan struct{})

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		close(started)
		<-release
		return nil
	}, mq, Opts{Slots: 10})

	written := make(chan error)
	go func() { written <- w.Write(context.Background(), newMsg(t)) }()
	<-started

	closed := make(chan error)
	go func() { closed <- w.Close(context.Background()) }()

	select {
	case <-closed:
		t.Fatal("Close returned while a synchronous write was still in flight")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	require.NoError(t, <-written)
	require.NoError(t, <-closed)
	assert.Equal(t, 0, mq.sentCount())
}

// Close racing with writes: every write either gets applied or goes over the queue, none is lost
// or blocked, and Close returns once the applied ones have finished.
func TestCloseRacingWithWrites(t *testing.T) {
	mq := &fakeMQ{}

	var applied atomic.Int32

	w := newWriter(t, func(_ uuid.UUID, _ string, payloads [][]byte) error {
		applied.Add(int32(len(payloads))) // nolint: gosec
		return nil
	}, mq, Opts{Slots: 4})

	const writers = 50

	var wg sync.WaitGroup

	var async atomic.Int32

	for i := range writers {
		wg.Go(func() {
			if i%2 == 0 {
				assert.NoError(t, w.Write(context.Background(), newMsg(t)))
				return
			}

			if !w.TryWriteAsync(context.Background(), newMsg(t)) {
				async.Add(1)
			}
		})
	}

	require.NoError(t, w.Close(context.Background()))
	wg.Wait()

	// TryWriteAsync calls that returned false are the caller's to send, so they appear in neither count.
	assert.Equal(t, int32(writers), applied.Load()+int32(mq.sentCount())+async.Load()) // nolint: gosec
}

func TestTryWriteAsyncLeavesEverythingToTheCallerWithoutACap(t *testing.T) {
	mq := &fakeMQ{}

	var applied atomic.Bool

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		applied.Store(true)
		return nil
	}, mq, Opts{Slots: -1})

	assert.False(t, w.TryWriteAsync(context.Background(), newMsg(t)))
	assert.False(t, applied.Load())
	assert.Equal(t, 0, mq.sentCount())
}

// A message whose batch failed, for example because another caller's message in it was bad, is
// applied on its own instead of going over the queue.
func TestFailedBatchMessageIsAppliedAlone(t *testing.T) {
	mq := &fakeMQ{}

	var calls atomic.Int32

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		if calls.Add(1) == 1 {
			return errors.New("invalid input syntax for type json")
		}

		return nil
	}, mq, Opts{Slots: 10})

	require.NoError(t, w.Write(context.Background(), newMsg(t)))
	assert.Equal(t, int32(2), calls.Load())
	assert.Equal(t, 0, mq.sentCount())
}

func TestMessageThatFailsAloneFallsBack(t *testing.T) {
	mq := &fakeMQ{}

	var calls atomic.Int32

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		calls.Add(1)
		return errors.New("invalid input syntax for type json")
	}, mq, Opts{Slots: 10})

	require.NoError(t, w.Write(context.Background(), newMsg(t)))
	assert.Equal(t, int32(2), calls.Load(), "one batched attempt and one alone")
	assert.Equal(t, 1, mq.sentCount())
}

// A real multi-message batch with one bad message in it: the healthy messages are applied on their
// own, and only the bad one goes over the queue.
func TestBadMessageInABatchOnlySendsItselfOverTheQueue(t *testing.T) {
	// One flush at a time per (tenant, message type), so the messages written while the first flush
	// is held pile up and go into the next flush together.
	orig := msgqueue.SUB_MAX_CONCURRENCY
	msgqueue.SUB_MAX_CONCURRENCY = 1
	t.Cleanup(func() { msgqueue.SUB_MAX_CONCURRENCY = orig })

	mq := &fakeMQ{}
	tenantID := uuid.New()
	gate := make(chan struct{})
	gateHeld := make(chan struct{})

	var (
		mu         sync.Mutex
		batchSizes []int
		applied    []string
	)

	msgWith := func(body string) *msgqueue.Message {
		msg, err := msgqueue.NewTenantMessage(tenantID, msgqueue.MsgIDTaskCompleted, false, true, map[string]string{"body": body})
		require.NoError(t, err)
		return msg
	}

	w := newWriter(t, func(_ uuid.UUID, _ string, payloads [][]byte) error {
		mu.Lock()
		batchSizes = append(batchSizes, len(payloads))
		first := len(batchSizes) == 1
		mu.Unlock()

		if first {
			close(gateHeld)
			<-gate
		}

		for _, p := range payloads {
			if strings.Contains(string(p), "bad") {
				return errors.New("invalid input syntax for type json")
			}
		}

		mu.Lock()
		for _, p := range payloads {
			applied = append(applied, string(p))
		}
		mu.Unlock()

		return nil
	}, mq, Opts{Slots: 10})

	var wg sync.WaitGroup

	wg.Go(func() { assert.NoError(t, w.Write(context.Background(), msgWith("gate"))) })
	<-gateHeld

	for _, body := range []string{"good-1", "bad", "good-2"} {
		wg.Go(func() { assert.NoError(t, w.Write(context.Background(), msgWith(body))) })
	}

	// Nothing reports when a message has landed in the buffer; this leaves the three writes time to
	// enqueue behind the held flush.
	time.Sleep(100 * time.Millisecond)
	close(gate)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	require.Contains(t, batchSizes, 3, "the three messages were not flushed as one batch: %v", batchSizes)
	require.Equal(t, 1, mq.sentCount())
	assert.Contains(t, string(mq.sent[0].Payloads[0]), "bad")
	assert.Len(t, applied, 3, "gate, good-1 and good-2 applied in-process: %v", applied)
}

func setFastBackoff(t *testing.T) {
	t.Helper()

	orig := fallbackRetryBackoff
	fallbackRetryBackoff = time.Millisecond
	t.Cleanup(func() { fallbackRetryBackoff = orig })
}
