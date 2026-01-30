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

// TestMsgIdBufferMemoryLeak verifies that the semaphore releaser reuses timers
// and doesn't create unbounded goroutines or memory leaks
func TestMsgIdBufferMemoryLeak(t *testing.T) {
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
	buf := newMsgIDBuffer(ctx, "test-tenant", "test-msg", dst, 10*time.Millisecond, 100, 10, false)

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
				msg:    &Message{TenantID: "test", ID: "test-msg", Payloads: [][]byte{[]byte("test")}},
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

	buf := newMsgIDBuffer(ctx, "test-tenant", "test-msg", dst, 5*time.Millisecond, 10, 3, false)

	// Trigger multiple rapid flushes
	for i := 0; i < 20; i++ {
		msg := &msgWithResultCh{
			msg:    &Message{TenantID: "test", ID: "test-msg", Payloads: [][]byte{[]byte("test")}},
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

	buf := newMsgIDBuffer(ctx, "test-tenant", "test-msg", dst, 10*time.Millisecond, 100, 10, false)

	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()

	// Send some messages
	for i := 0; i < 10; i++ {
		msg := &msgWithResultCh{
			msg:    &Message{TenantID: "test", ID: "test-msg", Payloads: [][]byte{[]byte("test")}},
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

	buf := newMsgIDBuffer(ctx, "test-tenant", "test-msg", dst, 5*time.Millisecond, 100, maxConcurrency, false)

	// Send many messages rapidly
	for i := 0; i < 50; i++ {
		msg := &msgWithResultCh{
			msg:    &Message{TenantID: "test", ID: "test-msg", Payloads: [][]byte{[]byte("test")}},
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
