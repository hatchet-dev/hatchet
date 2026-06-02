package msgqueue

import (
	"context"
	"os"
	"strconv"
	"time"
)

// nolint: staticcheck
func init() {
	if v := os.Getenv("SERVER_DEFAULT_BUFFER_FLUSH_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			PUB_FLUSH_INTERVAL = d
			SUB_FLUSH_INTERVAL = d
		}
	}
	if v := os.Getenv("SERVER_DEFAULT_BUFFER_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			PUB_BUFFER_SIZE = n
			SUB_BUFFER_SIZE = n
		}
	}
	if v := os.Getenv("SERVER_DEFAULT_BUFFER_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			PUB_MAX_CONCURRENCY = n
			SUB_MAX_CONCURRENCY = n
		}
	}
}

// bufferCore holds the shared concurrency machinery used by both the pub and sub
// per-(tenantId, msgId) inner buffers: a semaphore, its rate-limit releaser, a
// notifier for immediate flushes, and a capacity-triggered early-release signal.
type bufferCore struct {
	notifier         chan struct{}
	semaphore        chan struct{}
	semaphoreRelease chan time.Duration
	// capacityRelease interrupts the post-flush interval wait when the channel is
	// at capacity, triggering an immediate flush instead of waiting flushInterval.
	capacityRelease chan struct{}

	flushInterval         time.Duration
	bufferSize            int
	disableImmediateFlush bool
	// drainOnShutdown controls whether flush is called synchronously on ctx cancellation.
	drainOnShutdown bool
}

func newBufferCore(flushInterval time.Duration, bufferSize, maxConcurrency int, disableImmediateFlush, drainOnShutdown bool) bufferCore {
	return bufferCore{
		notifier:              make(chan struct{}),
		semaphore:             make(chan struct{}, maxConcurrency),
		semaphoreRelease:      make(chan time.Duration, maxConcurrency),
		capacityRelease:       make(chan struct{}, 1),
		flushInterval:         flushInterval,
		bufferSize:            bufferSize,
		disableImmediateFlush: disableImmediateFlush,
		drainOnShutdown:       drainOnShutdown,
	}
}

// startFlusher starts the ticker- and notifier-driven flush loop. flush is called
// as go flush() for ticker and notifier events; on shutdown it is called
// synchronously when drainOnShutdown is set.
func (c *bufferCore) startFlusher(ctx context.Context, flush func()) {
	go func() {
		ticker := time.NewTicker(c.flushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				if c.drainOnShutdown {
					flush()
				}
				return
			case <-ticker.C:
				go flush()
			case <-c.notifier:
				if !c.disableImmediateFlush {
					go flush()
				}
			}
		}
	}()
}

// startSemaphoreReleaser starts the goroutine that releases one semaphore slot
// after each flush's rate-limit delay. bufLen returns the current number of
// buffered messages and is checked after an early release to decide whether to
// self-trigger another flush.
func (c *bufferCore) startSemaphoreReleaser(ctx context.Context, bufLen func() int, flush func()) {
	go func() {
		timer := time.NewTimer(0)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case delay := <-c.semaphoreRelease:
				wasEarlyRelease := false
				if delay > 0 {
					timer.Reset(delay)
					select {
					case <-timer.C:
					case <-c.capacityRelease: // buffer hit capacity mid-interval
						wasEarlyRelease = true
					case <-ctx.Done():
						return
					}
				}
				<-c.semaphore
				// Only self-trigger on an early release — the goroutine that filled the
				// buffer is blocked before its notifier send, so nothing else will fire.
				if wasEarlyRelease && bufLen() > 0 {
					go flush()
				}
			}
		}
	}()
}

// drainN reads up to n items from ch without blocking. It returns as soon as the
// channel is empty, even if fewer than n items have been read.
func drainN[T any](ch <-chan T, n int) []T {
	items := make([]T, 0, n)
	for i := 0; i < n; i++ {
		select {
		case item := <-ch:
			items = append(items, item)
		default:
			return items
		}
	}
	return items
}
