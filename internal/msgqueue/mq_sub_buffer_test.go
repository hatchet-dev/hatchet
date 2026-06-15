package msgqueue

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

var testTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// TestSubBufferFlushesWhenFull verifies that when the channel is at capacity the
// capacityRelease mechanism breaks the post-flush interval wait and triggers an
// immediate second flush, without waiting the full 10s interval.
func TestSubBufferFlushesWhenFull(t *testing.T) {
	const bufSize = 5
	origSize := SUB_BUFFER_SIZE
	origInterval := SUB_FLUSH_INTERVAL
	SUB_BUFFER_SIZE = bufSize
	SUB_FLUSH_INTERVAL = 10 * time.Second
	defer func() {
		SUB_BUFFER_SIZE = origSize
		SUB_FLUSH_INTERVAL = origInterval
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The first dst call blocks until firstFlushRelease is closed, holding the semaphore
	// so we can fill the channel while it is locked.
	firstFlushStarted := make(chan struct{})
	firstFlushRelease := make(chan struct{})
	flushed := make(chan [][]byte, bufSize+2)

	var callCount atomic.Int32
	dst := func(_ uuid.UUID, _ string, payloads [][]byte) error {
		if callCount.Add(1) == 1 {
			close(firstFlushStarted)
			<-firstFlushRelease
		}
		flushed <- payloads
		return nil
	}

	buf := NewMQSubBuffer(TASK_PROCESSING_QUEUE, &mockMessageQueue{}, dst,
		WithFlushInterval(SUB_FLUSH_INTERVAL),
		WithBufferSize(bufSize),
		WithMaxConcurrency(1),
	)

	msg := &Message{TenantID: testTenantID, ID: "test-msg", Payloads: [][]byte{[]byte("p")}}

	// Start a handleMsg that triggers flush1 and holds the semaphore inside dst.
	go buf.handleMsg(ctx, msg) //nolint:errcheck
	<-firstFlushStarted        // semaphore is now held

	// Fill the channel to capacity. Each goroutine blocks on its result channel until flushed.
	var wg sync.WaitGroup
	for i := 0; i < bufSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = buf.handleMsg(ctx, msg)
		}()
	}

	// Give the goroutines time to enqueue (channel capacity = bufSize, all sends non-blocking).
	time.Sleep(50 * time.Millisecond)

	// One more handleMsg finds the channel at capacity, writes to capacityRelease, then blocks.
	var overflowWg sync.WaitGroup
	overflowWg.Add(1)
	go func() {
		defer overflowWg.Done()
		_ = buf.handleMsg(ctx, msg)
	}()

	// Release flush1. The semaphore releaser picks up the capacityRelease signal immediately
	// and triggers flush2 without waiting the full 10s.
	close(firstFlushRelease)
	<-flushed // discard flush1's payload

	// The bufSize buffered messages should complete via flush2, well within 2s.
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sub buffer did not flush all messages within 2s; flush interval is 10s — capacityRelease did not trigger")
	}
}

// TestMsgIdBufferMemoryLeak verifies that the semaphore releaser reuses timers
// and doesn't create unbounded goroutines or memory leaks
func TestMsgIdBufferMemoryLeak(t *testing.T) {
	const testBufSize = 100
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var processedCount atomic.Int64
	dst := func(tenantId uuid.UUID, msgId string, payloads [][]byte) error {
		processedCount.Add(1)
		// Simulate some processing time
		time.Sleep(1 * time.Millisecond)
		return nil
	}

	// Create a buffer
	buf := newMsgIDBuffer(ctx, testTenantID, "test-msg", dst, 10*time.Millisecond, testBufSize, 0, 10, false)

	// Force GC and get baseline
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	var baselineMemStats runtime.MemStats
	runtime.ReadMemStats(&baselineMemStats)
	baselineGoroutines := runtime.NumGoroutine()

	// Send many messages to trigger many flushes
	const numMessages = 1000
	var wg sync.WaitGroup
	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg := &msgWithResultCh{
				msg:    &Message{TenantID: testTenantID, ID: "test-msg", Payloads: [][]byte{[]byte("test")}},
				result: make(chan error, 1),
			}
			select {
			case buf.msgIdBufferCh <- msg:
				buf.notifier <- struct{}{}
			case <-time.After(100 * time.Millisecond):
				t.Error("timeout sending message")
			}
		}()
	}

	wg.Wait()

	// Wait for processing to complete
	time.Sleep(200 * time.Millisecond)

	// Force GC and check memory
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	var afterMemStats runtime.MemStats
	runtime.ReadMemStats(&afterMemStats)
	afterGoroutines := runtime.NumGoroutine()

	// Verify we processed messages
	if processedCount.Load() == 0 {
		t.Error("No messages were processed")
	}

	// Verify goroutine count didn't explode
	// We should have approximately the same number of goroutines
	// (baseline + 2 for the buffer: startFlusher + startSemaphoreReleaser)
	goroutineDiff := afterGoroutines - baselineGoroutines
	if goroutineDiff > 5 {
		t.Errorf("Too many goroutines created: baseline=%d, after=%d, diff=%d (expected <=5)",
			baselineGoroutines, afterGoroutines, goroutineDiff)
	}

	// Verify memory didn't grow excessively
	// With 1000 flushes, if we were creating goroutines+timers for each,
	// we'd see significant memory growth (multiple MB)
	memGrowthMB := float64(afterMemStats.Alloc-baselineMemStats.Alloc) / 1024 / 1024
	if memGrowthMB > 5 {
		t.Errorf("Excessive memory growth: %.2f MB (expected <5MB)", memGrowthMB)
	}

	t.Logf("Processed %d messages", processedCount.Load())
	t.Logf("Goroutines: baseline=%d, after=%d, diff=%d", baselineGoroutines, afterGoroutines, goroutineDiff)
	t.Logf("Memory growth: %.2f MB", memGrowthMB)
}

// TestSemaphoreReleaserReusesTimer verifies the semaphore releaser properly reuses one timer
func TestSemaphoreReleaserReusesTimer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var flushCount atomic.Int64
	dst := func(tenantId uuid.UUID, msgId string, payloads [][]byte) error {
		flushCount.Add(1)
		return nil
	}

	buf := newMsgIDBuffer(ctx, testTenantID, "test-msg", dst, 5*time.Millisecond, 10, 0, 3, false)

	// Trigger multiple rapid flushes
	for i := 0; i < 20; i++ {
		msg := &msgWithResultCh{
			msg:    &Message{TenantID: testTenantID, ID: "test-msg", Payloads: [][]byte{[]byte("test")}},
			result: make(chan error, 1),
		}
		buf.msgIdBufferCh <- msg
		buf.notifier <- struct{}{}
		time.Sleep(2 * time.Millisecond)
	}

	// Wait for flushes to complete
	time.Sleep(100 * time.Millisecond)

	// Verify we had multiple flushes (showing rate limiting works)
	if flushCount.Load() < 5 {
		t.Errorf("Expected at least 5 flushes, got %d", flushCount.Load())
	}

	t.Logf("Completed %d flushes", flushCount.Load())
}

// TestBufferCleanupOnContextCancel verifies proper cleanup when context is cancelled
func TestBufferCleanupOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	dst := func(tenantId uuid.UUID, msgId string, payloads [][]byte) error {
		return nil
	}

	buf := newMsgIDBuffer(ctx, testTenantID, "test-msg", dst, 10*time.Millisecond, 100, 0, 10, false)

	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()

	// Send some messages
	for i := 0; i < 10; i++ {
		msg := &msgWithResultCh{
			msg:    &Message{TenantID: testTenantID, ID: "test-msg", Payloads: [][]byte{[]byte("test")}},
			result: make(chan error, 1),
		}
		buf.msgIdBufferCh <- msg
		buf.notifier <- struct{}{}
	}

	// Cancel context
	cancel()

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	afterGoroutines := runtime.NumGoroutine()

	// Goroutines should be cleaned up (back to baseline or close to it)
	goroutineDiff := afterGoroutines - baselineGoroutines
	if goroutineDiff > 2 {
		t.Errorf("Goroutines not cleaned up properly: baseline=%d, after=%d, diff=%d",
			baselineGoroutines, afterGoroutines, goroutineDiff)
	}

	t.Logf("Cleanup successful: baseline=%d, after=%d", baselineGoroutines, afterGoroutines)
}

// TestConcurrentFlushesRateLimited verifies semaphore properly limits concurrent flushes
func TestConcurrentFlushesRateLimited(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const maxConcurrency = 3
	var currentConcurrent atomic.Int32
	var maxObservedConcurrent atomic.Int32

	dst := func(tenantId uuid.UUID, msgId string, payloads [][]byte) error {
		current := currentConcurrent.Add(1)
		defer currentConcurrent.Add(-1)

		// Track max concurrent
		for {
			max := maxObservedConcurrent.Load()
			if current <= max || maxObservedConcurrent.CompareAndSwap(max, current) {
				break
			}
		}

		// Simulate work
		time.Sleep(20 * time.Millisecond)
		return nil
	}

	buf := newMsgIDBuffer(ctx, testTenantID, "test-msg", dst, 5*time.Millisecond, 100, 0, maxConcurrency, false)

	// Send many messages rapidly
	for i := 0; i < 50; i++ {
		msg := &msgWithResultCh{
			msg:    &Message{TenantID: testTenantID, ID: "test-msg", Payloads: [][]byte{[]byte("test")}},
			result: make(chan error, 1),
		}
		buf.msgIdBufferCh <- msg
		buf.notifier <- struct{}{}
		time.Sleep(1 * time.Millisecond)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	maxConcurrent := maxObservedConcurrent.Load()
	if maxConcurrent > int32(maxConcurrency) {
		t.Errorf("Concurrency limit violated: max observed=%d, limit=%d", maxConcurrent, maxConcurrency)
	}

	t.Logf("Max concurrent flushes observed: %d (limit: %d)", maxConcurrent, maxConcurrency)
}

// waitForPayloads blocks until got >= want or the timeout elapses.
func waitForPayloads(t *testing.T, mu *sync.Mutex, got *int, want int, timeout time.Duration) {
	t.Helper()

	deadline := time.After(timeout)

	for {
		mu.Lock()
		cur := *got
		mu.Unlock()

		if cur >= want {
			return
		}

		select {
		case <-deadline:
			t.Fatalf("timed out waiting for payloads: got %d/%d", cur, want)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// TestFlushRespectsMaxBytes verifies that maxFlushBytes bounds the total
// payload bytes drained into a single flush (i.e. a single dst/bulk-write call)
// while the count cap (bufferSize) is intentionally left high.
func TestFlushRespectsMaxBytes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const payloadSize = 1024
	const maxFlushBytes = 4096

	var mu sync.Mutex
	var flushSizes []int
	var totalPayloads int

	dst := func(tenantId uuid.UUID, msgId string, payloads [][]byte) error {
		size := 0
		for _, p := range payloads {
			size += len(p)
		}

		mu.Lock()
		flushSizes = append(flushSizes, size)
		totalPayloads += len(payloads)
		mu.Unlock()

		return nil
	}

	// bufferSize is large so the byte cap (not the count cap) bounds each flush.
	// maxConcurrency=1 serializes flushes so per-flush sizes are deterministic.
	buf := newMsgIDBuffer(ctx, testTenantID, "test-msg", dst, 5*time.Millisecond, 1000, maxFlushBytes, 1, false)

	const numMessages = 40
	for i := 0; i < numMessages; i++ {
		msg := &msgWithResultCh{
			msg:    &Message{TenantID: testTenantID, ID: "test-msg", Payloads: [][]byte{make([]byte, payloadSize)}},
			result: make(chan error, 1),
		}
		buf.msgIdBufferCh <- msg
		buf.notifier <- struct{}{}
	}

	waitForPayloads(t, &mu, &totalPayloads, numMessages, 2*time.Second)

	mu.Lock()
	defer mu.Unlock()

	if totalPayloads != numMessages {
		t.Errorf("expected %d payloads processed, got %d", numMessages, totalPayloads)
	}

	for i, size := range flushSizes {
		if size > maxFlushBytes {
			t.Errorf("flush %d carried %d bytes, exceeds cap %d", i, size, maxFlushBytes)
		}
	}

	t.Logf("flushes=%d sizes=%v", len(flushSizes), flushSizes)
}

// TestFlushTakesAtLeastOneOversizedMessage verifies that a message whose payload
// alone exceeds maxFlushBytes still drains (one per flush) rather than wedging
// the buffer.
func TestFlushTakesAtLeastOneOversizedMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const payloadSize = 4096
	const maxFlushBytes = 512 // smaller than a single payload

	var mu sync.Mutex
	var flushSizes []int
	var totalPayloads int

	dst := func(tenantId uuid.UUID, msgId string, payloads [][]byte) error {
		size := 0
		for _, p := range payloads {
			size += len(p)
		}

		mu.Lock()
		flushSizes = append(flushSizes, size)
		totalPayloads += len(payloads)
		mu.Unlock()

		return nil
	}

	buf := newMsgIDBuffer(ctx, testTenantID, "test-msg", dst, 5*time.Millisecond, 1000, maxFlushBytes, 1, false)

	const numMessages = 5
	for i := 0; i < numMessages; i++ {
		msg := &msgWithResultCh{
			msg:    &Message{TenantID: testTenantID, ID: "test-msg", Payloads: [][]byte{make([]byte, payloadSize)}},
			result: make(chan error, 1),
		}
		buf.msgIdBufferCh <- msg
		buf.notifier <- struct{}{}
	}

	waitForPayloads(t, &mu, &totalPayloads, numMessages, 2*time.Second)

	mu.Lock()
	defer mu.Unlock()

	if totalPayloads != numMessages {
		t.Errorf("expected %d payloads processed, got %d", numMessages, totalPayloads)
	}

	for i, size := range flushSizes {
		if size != payloadSize {
			t.Errorf("flush %d carried %d bytes, expected exactly one %d-byte message", i, size, payloadSize)
		}
	}

	t.Logf("flushes=%d sizes=%v", len(flushSizes), flushSizes)
}
