package msgqueue

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

// mockMessageQueue satisfies MessageQueue for tests. Only SendMessage is wired up.
type mockMessageQueue struct {
	sendMessageFn func(ctx context.Context, q Queue, msg *Message) error
}

func (m *mockMessageQueue) Clone() (func() error, MessageQueue, error) {
	return func() error { return nil }, m, nil
}
func (m *mockMessageQueue) SetQOS(_ int) {}
func (m *mockMessageQueue) SendMessage(ctx context.Context, q Queue, msg *Message) error {
	if m.sendMessageFn != nil {
		return m.sendMessageFn(ctx, q, msg)
	}
	return nil
}
func (m *mockMessageQueue) Subscribe(_ Queue, _ MsgHandler, _ MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}
func (m *mockMessageQueue) IsReady() bool { return true }

// TestPubBufferFlushesWhenFull verifies that when the channel is at capacity the
// capacityRelease mechanism breaks the post-flush interval wait and triggers an
// immediate second flush, without waiting the full 10s interval.
func TestPubBufferFlushesWhenFull(t *testing.T) {
	const bufSize = 5
	origSize := PUB_BUFFER_SIZE
	origInterval := PUB_FLUSH_INTERVAL
	origConcurrency := PUB_MAX_CONCURRENCY
	PUB_BUFFER_SIZE = bufSize
	PUB_FLUSH_INTERVAL = 10 * time.Second
	PUB_MAX_CONCURRENCY = 1
	defer func() {
		PUB_BUFFER_SIZE = origSize
		PUB_FLUSH_INTERVAL = origInterval
		PUB_MAX_CONCURRENCY = origConcurrency
	}()

	// The first SendMessage call blocks until firstFlushRelease is closed, which holds the
	// semaphore so we can fill the channel while it is locked.
	firstFlushStarted := make(chan struct{})
	firstFlushRelease := make(chan struct{})
	received := make(chan *Message, bufSize+2)

	var callCount atomic.Int32
	mq := &mockMessageQueue{
		sendMessageFn: func(_ context.Context, _ Queue, msg *Message) error {
			if callCount.Add(1) == 1 {
				close(firstFlushStarted)
				<-firstFlushRelease
			}
			received <- msg
			return nil
		},
	}

	buf := NewMQPubBuffer(mq)
	defer buf.Stop()

	ctx := context.Background()
	msg := &Message{TenantID: testTenantID, ID: "test-msg", Payloads: [][]byte{[]byte("p")}}

	// Start a Pub that will trigger flush1 and hold the semaphore inside SendMessage.
	go buf.Pub(ctx, TASK_PROCESSING_QUEUE, msg, false)
	<-firstFlushStarted // semaphore is now held; interval wait hasn't started yet

	// Fill the channel to capacity with sequential Pubs (non-blocking: channel is empty
	// because flush1's read loop ran before these sends, and flush1 is blocked in SendMessage).
	for i := 0; i < bufSize; i++ {
		_ = buf.Pub(ctx, TASK_PROCESSING_QUEUE, msg, false)
	}

	// One more Pub finds the channel at capacity, writes to capacityRelease, then blocks on
	// the channel send until a flush drains it.
	overflowDone := make(chan struct{})
	go func() {
		defer close(overflowDone)
		_ = buf.Pub(ctx, TASK_PROCESSING_QUEUE, msg, false)
	}()

	// Release flush1. The semaphore releaser will immediately pick up the capacityRelease
	// signal (already buffered) and trigger flush2 without waiting the full 10s.
	close(firstFlushRelease)
	<-received // discard flush1's single message

	// The bufSize buffered messages should now appear via flush2, well within 2s.
	var total int
	deadline := time.After(2 * time.Second)
	for total < bufSize {
		select {
		case m := <-received:
			total += len(m.Payloads)
		case <-deadline:
			t.Fatalf("pub buffer flushed %d/%d buffered payloads within 2s; flush interval is 10s — capacityRelease did not trigger", total, bufSize)
		}
	}

	// Flush2 drained the channel, so the overflow goroutine should have unblocked.
	select {
	case <-overflowDone:
	case <-time.After(2 * time.Second):
		t.Error("overflow Pub did not unblock after buffer was drained")
	}
}

func countingMQ(sent *atomic.Int32) *mockMessageQueue {
	return &mockMessageQueue{sendMessageFn: func(_ context.Context, _ Queue, msg *Message) error {
		sent.Add(int32(len(msg.Payloads))) // nolint: gosec
		return nil
	}}
}

func newTestMessage(t *testing.T, tenantID uuid.UUID) *Message {
	t.Helper()

	msg, err := NewTenantMessage(tenantID, MsgIDTaskAcked, false, true, map[string]string{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}

	return msg
}

// Messages enqueued right after a flush sit out the flush interval, and the flusher's own shutdown
// flush skips them; Stop must still publish them before it returns.
func TestPubBufferStopPublishesEverythingBeforeReturning(t *testing.T) {
	var sent atomic.Int32

	b := NewMQPubBuffer(countingMQ(&sent))
	tenantID := uuid.New()

	if err := b.Pub(context.Background(), TASK_PROCESSING_QUEUE, newTestMessage(t, tenantID), true); err != nil {
		t.Fatal(err)
	}

	for range 3 {
		if err := b.Pub(context.Background(), TASK_PROCESSING_QUEUE, newTestMessage(t, tenantID), false); err != nil {
			t.Fatal(err)
		}
	}

	b.Stop()

	if got := sent.Load(); got != 4 {
		t.Fatalf("expected all 4 messages published when Stop returned, got %d", got)
	}
}

func TestPubBufferPubAfterStopPublishesDirectly(t *testing.T) {
	var sent atomic.Int32

	b := NewMQPubBuffer(countingMQ(&sent))
	tenantID := uuid.New()

	// the tenant's buffer exists before Stop, so its flusher has exited by the time of the second Pub
	if err := b.Pub(context.Background(), TASK_PROCESSING_QUEUE, newTestMessage(t, tenantID), true); err != nil {
		t.Fatal(err)
	}

	b.Stop()

	done := make(chan error, 1)
	go func() {
		done <- b.Pub(context.Background(), TASK_PROCESSING_QUEUE, newTestMessage(t, tenantID), true)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Pub after Stop blocked")
	}

	if got := sent.Load(); got != 2 {
		t.Fatalf("expected 2 messages published, got %d", got)
	}
}

// Pub calls racing with Stop either land in a buffer that Stop drains or publish themselves; none is
// lost or left blocked.
func TestPubBufferPubRacingWithStop(t *testing.T) {
	var sent atomic.Int32

	b := NewMQPubBuffer(countingMQ(&sent))

	const pubs = 200

	// a handful of tenants, so messages share buffers and flush windows
	tenants := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}

	var wg sync.WaitGroup

	for i := range pubs {
		wg.Go(func() {
			tenantID := tenants[i%len(tenants)]
			if err := b.Pub(context.Background(), TASK_PROCESSING_QUEUE, newTestMessage(t, tenantID), i%3 == 0); err != nil {
				t.Error(err)
			}
		})
	}

	b.Stop()
	wg.Wait()

	if got := sent.Load(); got != pubs {
		t.Fatalf("expected %d messages published, got %d", pubs, got)
	}
}
