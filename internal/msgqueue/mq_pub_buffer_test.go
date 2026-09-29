package msgqueue

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
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
			t.Fatalf("pub buffer flushed %d/%d buffered payloads within 2s; flush interval is 10s, capacityRelease did not trigger", total, bufSize)
		}
	}

	// Flush2 drained the channel, so the overflow goroutine should have unblocked.
	select {
	case <-overflowDone:
	case <-time.After(2 * time.Second):
		t.Error("overflow Pub did not unblock after buffer was drained")
	}
}

// syncBuffer is a goroutine-safe io.Writer for capturing log output from the
// buffer's flush goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestPubBufferLogsDroppedBatchOnce verifies that a failed flush is published
// exactly once (no replay, since the backend may already have accepted it), is
// dropped with exactly one error log line carrying enough context to pin it,
// and hands the error to a waiter while a wait=false caller sees nil.
func TestPubBufferLogsDroppedBatchOnce(t *testing.T) {
	var callCount atomic.Int32
	mq := &mockMessageQueue{
		sendMessageFn: func(_ context.Context, _ Queue, _ *Message) error {
			callCount.Add(1)
			return errors.New("broker unavailable")
		},
	}

	logs := &syncBuffer{}
	l := zerolog.New(logs)

	buf := NewMQPubBuffer(mq, WithPubLogger(&l))
	defer buf.Stop()

	msg := &Message{TenantID: testTenantID, ID: "test-msg", Payloads: [][]byte{[]byte("p")}}

	if err := buf.Pub(context.Background(), OLAP_QUEUE, msg, false); err != nil {
		t.Fatalf("wait=false Pub should not surface the publish error, got %v", err)
	}

	// the drop line is written after the single publish attempt, so it doubles
	// as the flush-completion signal
	deadline := time.After(2 * time.Second)
	for !strings.Contains(logs.String(), "dropping buffered message") {
		select {
		case <-deadline:
			t.Fatalf("no drop log line within 2s, got:\n%s", logs.String())
		case <-time.After(time.Millisecond):
		}
	}

	if got := callCount.Load(); got != 1 {
		t.Fatalf("expected exactly one publish attempt, got %d", got)
	}

	out := logs.String()

	if got := strings.Count(out, "dropping buffered message"); got != 1 {
		t.Fatalf("expected exactly one drop log line, got %d in:\n%s", got, out)
	}

	for _, want := range []string{`"level":"error"`, `"queue":"olap_queue_v2"`, `"message_id":"test-msg"`, `"num_payloads":1`, `"num_waiters":0`, `"tenant_id":"` + testTenantID.String() + `"`, `"error":"broker unavailable"`} {
		if !strings.Contains(out, want) {
			t.Errorf("drop log should carry %s, got:\n%s", want, out)
		}
	}

	// a waiter is the one caller that can see the failure directly
	if err := buf.Pub(context.Background(), OLAP_QUEUE, msg, true); err == nil || err.Error() != "broker unavailable" {
		t.Fatalf("waiter should receive the publish error, got %v", err)
	}

	if got := callCount.Load(); got != 2 {
		t.Fatalf("expected the waiter's batch to be published exactly once more, got %d total attempts", got)
	}
}
