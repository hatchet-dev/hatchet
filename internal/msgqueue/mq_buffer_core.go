package msgqueue

import (
	"context"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// BUFFER_IDLE_TIMEOUT is how long a per-(tenantId, msgId) buffer can go
// without an enqueue before it is evicted from its buffer map and its
// goroutines are stopped.
//
// nolint: staticcheck
var BUFFER_IDLE_TIMEOUT = 5 * time.Minute

// nolint: staticcheck
func init() {
	if v := os.Getenv("SERVER_DEFAULT_BUFFER_FLUSH_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			PUB_FLUSH_INTERVAL = d
			SUB_FLUSH_INTERVAL = d
		}
	}
	if v := os.Getenv("SERVER_DEFAULT_BUFFER_IDLE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			BUFFER_IDLE_TIMEOUT = d
		}
	}
	if v := os.Getenv("SERVER_DEFAULT_BUFFER_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			PUB_BUFFER_SIZE = n
			SUB_BUFFER_SIZE = n
		}
	}
	if v := os.Getenv("SERVER_DEFAULT_BUFFER_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
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

	// evictMu guards closed: producers hold it shared for the duration of an
	// enqueue, the evictor holds it exclusively to close the buffer.
	evictMu    sync.RWMutex
	closed     bool
	lastActive atomic.Int64 // unix nanos of the last enqueue
	// stop cancels the buffer's flusher and semaphore-releaser goroutines.
	stop context.CancelFunc
}

func newBufferCore(flushInterval time.Duration, bufferSize, maxConcurrency int, disableImmediateFlush, drainOnShutdown bool) *bufferCore {
	c := &bufferCore{
		notifier:              make(chan struct{}),
		semaphore:             make(chan struct{}, maxConcurrency),
		semaphoreRelease:      make(chan time.Duration, maxConcurrency),
		capacityRelease:       make(chan struct{}, 1),
		flushInterval:         flushInterval,
		bufferSize:            bufferSize,
		disableImmediateFlush: disableImmediateFlush,
		drainOnShutdown:       drainOnShutdown,
	}
	c.lastActive.Store(time.Now().UnixNano())
	return c
}

// tryAcquire registers an enqueue against the buffer, returning false if the
// buffer has been evicted, in which case the caller must load or create a
// fresh buffer. On success the caller must call release once it has finished
// enqueueing.
func (c *bufferCore) tryAcquire() bool {
	c.evictMu.RLock()

	if c.closed {
		c.evictMu.RUnlock()
		return false
	}

	c.lastActive.Store(time.Now().UnixNano())

	return true
}

func (c *bufferCore) release() {
	c.evictMu.RUnlock()
}

// tryEvict marks the buffer closed if it has not seen an enqueue for at least
// idleTimeout. Once closed no enqueue can acquire the buffer again, so it is
// safe to remove it from the buffer map and stop its goroutines.
func (c *bufferCore) tryEvict(idleTimeout time.Duration) bool {
	c.evictMu.Lock()
	defer c.evictMu.Unlock()

	if c.closed || time.Since(time.Unix(0, c.lastActive.Load())) < idleTimeout {
		return false
	}

	c.closed = true

	return true
}

// startFlusher starts the ticker- and notifier-driven flush loop. flush is called
// as go flush() for ticker and notifier events; on shutdown it is called
// synchronously when drainOnShutdown is set.
//
// Buffers are created per (tenantId, msgId) and never removed, so the flusher
// parks with the ticker stopped once the buffer is empty. Otherwise every
// buffer ever created keeps waking up each flushInterval forever, which
// accumulates CPU on long-running processes as tenants come and go. Producers
// always send on notifier after enqueueing, so a parked flusher is guaranteed
// to be woken by the next message.
func (c *bufferCore) startFlusher(ctx context.Context, bufLen func() int, flush func()) {
	go func() {
		ticker := time.NewTicker(c.flushInterval)
		defer ticker.Stop()

		idle := false

		for {
			if idle {
				select {
				case <-ctx.Done():
					if c.drainOnShutdown {
						flush()
					}
					return
				case <-c.notifier:
					idle = false
					ticker.Reset(c.flushInterval)
					if !c.disableImmediateFlush {
						go flush()
					}
				}

				continue
			}

			select {
			case <-ctx.Done():
				if c.drainOnShutdown {
					flush()
				}
				return
			case <-ticker.C:
				if bufLen() == 0 {
					ticker.Stop()
					idle = true
					continue
				}

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
		<-timer.C // drain initial tick so the first Reset doesn't wake immediately
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
						if !timer.Stop() {
							select {
							case <-timer.C:
							default:
							}
						}
					case <-ctx.Done():
						return
					}
				} else {
					select {
					case <-c.capacityRelease:
						wasEarlyRelease = true
					default:
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
