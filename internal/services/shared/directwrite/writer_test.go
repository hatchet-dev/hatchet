package directwrite

import (
	"context"
	"errors"
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

func TestWriteAsyncFallsBackOnHandlerError(t *testing.T) {
	mq := &fakeMQ{}

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		return errors.New("deadlock detected")
	}, mq, Opts{Slots: 10})

	w.WriteAsync(context.Background(), newMsg(t))

	require.Eventually(t, func() bool { return mq.sentCount() == 1 }, 5*time.Second, 10*time.Millisecond)
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

func setFastBackoff(t *testing.T) {
	t.Helper()

	orig := fallbackRetryBackoff
	fallbackRetryBackoff = time.Millisecond
	t.Cleanup(func() { fallbackRetryBackoff = orig })
}
