package msgqueue

import (
	"context"
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
func (m *mockMessageQueue) Subscribe(_ Queue, _ AckHook, _ AckHook) (func() error, error) {
	return func() error { return nil }, nil
}
func (m *mockMessageQueue) RegisterTenant(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockMessageQueue) IsReady() bool                                       { return true }

// TestPubBufferFlushesWhenFull verifies that filling the buffer triggers a flush via the
// notifier path without waiting for the flush interval timer. A 10s flush interval is used
// so the test would time out if only the ticker triggered the flush.
func TestPubBufferFlushesWhenFull(t *testing.T) {
	const bufSize = 5
	origSize := PUB_BUFFER_SIZE
	origInterval := PUB_FLUSH_INTERVAL
	origConcurrency := PUB_MAX_CONCURRENCY
	PUB_BUFFER_SIZE = bufSize
	PUB_FLUSH_INTERVAL = 10 * time.Second
	PUB_MAX_CONCURRENCY = bufSize // one slot per message so each notifier signal can flush independently
	defer func() {
		PUB_BUFFER_SIZE = origSize
		PUB_FLUSH_INTERVAL = origInterval
		PUB_MAX_CONCURRENCY = origConcurrency
	}()

	// Capacity bufSize so SendMessage never blocks regardless of batching.
	received := make(chan *Message, bufSize)
	mq := &mockMessageQueue{
		sendMessageFn: func(_ context.Context, _ Queue, msg *Message) error {
			received <- msg
			return nil
		},
	}

	buf := NewMQPubBuffer(mq)
	defer buf.Stop()

	// Enqueue bufSize messages concurrently (fire-and-forget).
	for i := 0; i < bufSize; i++ {
		go func() {
			_ = buf.Pub(context.Background(), TASK_PROCESSING_QUEUE, &Message{
				TenantID: testTenantID,
				ID:       "test-msg",
				Payloads: [][]byte{[]byte("p")},
			}, false)
		}()
	}

	// Collect all published payloads from SendMessage calls. The deadline fires if any
	// message is still waiting after 2s — which only happens if it relies on the 10s timer.
	deadline := time.After(2 * time.Second)
	var total int
	for total < bufSize {
		select {
		case msg := <-received:
			total += len(msg.Payloads)
		case <-deadline:
			t.Fatalf("pub buffer flushed %d/%d payloads within 2s; flush interval is 10s, so the timer was not the trigger", total, bufSize)
		}
	}
}
