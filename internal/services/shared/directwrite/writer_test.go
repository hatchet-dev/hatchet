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

type fakeMQ struct {
	mu   sync.Mutex
	sent []*msgqueue.Message
	err  error
}

func (m *fakeMQ) Clone() (func() error, msgqueue.MessageQueue, error) {
	return func() error { return nil }, m, nil
}
func (m *fakeMQ) SetQOS(int) {}
func (m *fakeMQ) SendMessage(_ context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if q.Name() != msgqueue.TASK_PROCESSING_QUEUE.Name() {
		return errors.New("unexpected queue")
	}

	m.sent = append(m.sent, msg)
	return m.err
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
	}, mq, Opts{Slots: 10, Fallback: true})

	require.NoError(t, w.Write(context.Background(), newMsg(t)))
	assert.Equal(t, int32(1), handled.Load())
	assert.Equal(t, 0, mq.sentCount())
}

func TestWriteFallsBackOnHandlerError(t *testing.T) {
	mq := &fakeMQ{}

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		return errors.New("deadlock detected")
	}, mq, Opts{Slots: 10, Fallback: true})

	require.NoError(t, w.Write(context.Background(), newMsg(t)))
	assert.Equal(t, 1, mq.sentCount())
}

func TestWriteReturnsHandlerErrorWithoutFallback(t *testing.T) {
	mq := &fakeMQ{}

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		return errors.New("deadlock detected")
	}, mq, Opts{Slots: 10, Fallback: false})

	require.Error(t, w.Write(context.Background(), newMsg(t)))
	assert.Equal(t, 0, mq.sentCount())
}

func TestWriteFallsBackWhenSlotsAreTaken(t *testing.T) {
	mq := &fakeMQ{}
	release := make(chan struct{})
	started := make(chan struct{})

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		close(started)
		<-release
		return nil
	}, mq, Opts{Slots: 1, Fallback: false})

	done := make(chan error)
	go func() { done <- w.Write(context.Background(), newMsg(t)) }()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first write was never handled")
	}

	// The slot cap applies even with fallback disabled, which only covers handler errors.
	require.NoError(t, w.Write(context.Background(), newMsg(t)))
	assert.Equal(t, 1, mq.sentCount())

	close(release)
	require.NoError(t, <-done)
	assert.Equal(t, 1, mq.sentCount())
}

func TestWriteAsyncFallsBackEvenWithFallbackDisabled(t *testing.T) {
	mq := &fakeMQ{}

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		return errors.New("deadlock detected")
	}, mq, Opts{Slots: 10, Fallback: false})

	w.WriteAsync(context.Background(), newMsg(t))

	require.Eventually(t, func() bool { return mq.sentCount() == 1 }, 5*time.Second, 10*time.Millisecond)
}

func TestWriteReturnsQueueErrorOnFallback(t *testing.T) {
	mq := &fakeMQ{err: errors.New("broker down")}

	w := newWriter(t, func(uuid.UUID, string, [][]byte) error {
		return errors.New("deadlock detected")
	}, mq, Opts{Slots: 10, Fallback: true})

	require.Error(t, w.Write(context.Background(), newMsg(t)))
}
