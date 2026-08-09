package msgqueue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func waitForFlushCount(t *testing.T, flushCount *atomic.Int64, min int64) {
	t.Helper()

	deadline := time.After(time.Second)

	for flushCount.Load() < min {
		select {
		case <-deadline:
			t.Fatalf("expected at least %d flushes, got %d", min, flushCount.Load())
		case <-time.After(time.Millisecond):
		}
	}
}

// TestFlusherParksWhenIdle verifies the flush loop stops ticking while the buffer
// is empty and resumes when a producer signals the notifier. Buffers are never
// removed from the per-(tenantId, msgId) maps, so a flusher that keeps ticking
// while idle accumulates CPU forever on long-running processes.
func TestFlusherParksWhenIdle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newBufferCore(5*time.Millisecond, 10, 3, false, false)

	var bufLen atomic.Int64
	var flushCount atomic.Int64

	c.startFlusher(ctx, func() int { return int(bufLen.Load()) }, func() {
		flushCount.Add(1)
	})

	// The first tick observes an empty buffer and parks; no flushes should fire.
	time.Sleep(50 * time.Millisecond)

	if got := flushCount.Load(); got != 0 {
		t.Fatalf("expected no flushes while idle, got %d", got)
	}

	bufLen.Store(1)
	c.notifier <- struct{}{}

	waitForFlushCount(t, &flushCount, 1)

	// Drain the buffer and confirm the flusher parks again.
	bufLen.Store(0)
	time.Sleep(50 * time.Millisecond)

	parked := flushCount.Load()
	time.Sleep(50 * time.Millisecond)

	if got := flushCount.Load(); got != parked {
		t.Fatalf("expected flusher to park after buffer drained, flushes went %d -> %d", parked, got)
	}

	// A parked flusher must wake again for subsequent messages.
	bufLen.Store(1)
	c.notifier <- struct{}{}

	waitForFlushCount(t, &flushCount, parked+1)
}

// TestSubBufferEvictsIdleBuffers verifies that buffers which have not seen a
// message for BUFFER_IDLE_TIMEOUT are removed from the buffers map and that a
// later message for the same key is processed through a freshly created buffer.
func TestSubBufferEvictsIdleBuffers(t *testing.T) {
	oldTimeout := BUFFER_IDLE_TIMEOUT
	BUFFER_IDLE_TIMEOUT = 20 * time.Millisecond
	defer func() { BUFFER_IDLE_TIMEOUT = oldTimeout }()

	var handler MsgHandler
	sub := func(preAck MsgHandler, postAck MsgHandler) (func() error, error) {
		handler = preAck
		return func() error { return nil }, nil
	}

	var processed atomic.Int64
	dst := func(tenantId uuid.UUID, msgId string, payloads [][]byte) error {
		processed.Add(1)
		return nil
	}

	buf := NewSubBufferFromSubscribe(sub, dst)

	cleanup, err := buf.Start()
	if err != nil {
		t.Fatalf("could not start sub buffer: %v", err)
	}
	defer cleanup() // nolint: errcheck

	send := func() {
		msg := &Message{TenantID: testTenantID, ID: "test-msg", Payloads: [][]byte{[]byte("p")}}
		if err := handler(msg); err != nil {
			t.Fatalf("could not handle message: %v", err)
		}
	}

	send()

	if got := buf.buffers.Len(); got != 1 {
		t.Fatalf("expected 1 buffer after first message, got %d", got)
	}

	deadline := time.After(time.Second)
	for buf.buffers.Len() != 0 {
		select {
		case <-deadline:
			t.Fatal("idle buffer was not evicted")
		case <-time.After(5 * time.Millisecond):
		}
	}

	// a message after eviction must create a fresh buffer and still be processed
	send()

	if got := processed.Load(); got != 2 {
		t.Fatalf("expected 2 processed flushes, got %d", got)
	}
}

// TestParkedFlusherRestartsTickerWithImmediateFlushDisabled verifies that when
// immediate flushes are disabled, waking from the parked state restarts the
// ticker so the buffered message is still flushed on the next interval.
func TestParkedFlusherRestartsTickerWithImmediateFlushDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := newBufferCore(5*time.Millisecond, 10, 3, true, false)

	var bufLen atomic.Int64
	var flushCount atomic.Int64

	c.startFlusher(ctx, func() int { return int(bufLen.Load()) }, func() {
		flushCount.Add(1)
	})

	// Let the flusher park on an empty buffer.
	time.Sleep(50 * time.Millisecond)

	if got := flushCount.Load(); got != 0 {
		t.Fatalf("expected no flushes while idle, got %d", got)
	}

	// With immediate flush disabled the notifier only rearms the ticker, so the
	// flush must come from a tick.
	bufLen.Store(1)
	c.notifier <- struct{}{}

	waitForFlushCount(t, &flushCount, 1)
}
