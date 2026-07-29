package dispatcher

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// refreshTimeoutFlushInterval caps how often a given task's buffered timeout
// refreshes are flushed to Postgres on this engine process.
const refreshTimeoutFlushInterval = 500 * time.Millisecond

type refreshTimeoutKey struct {
	tenantId       uuid.UUID
	taskExternalId uuid.UUID
}

func (k refreshTimeoutKey) String() string {
	return k.tenantId.String() + ":" + k.taskExternalId.String()
}

type refreshTimeoutBuffer struct {
	mu      sync.Mutex
	entries map[refreshTimeoutKey]*refreshTimeoutBufEntry
}

type refreshTimeoutBufEntry struct {
	pending       time.Duration
	lastTimeoutAt time.Time
	lastFlushAt   time.Time
	timer         *time.Timer
}

func newRefreshTimeoutBuffer() *refreshTimeoutBuffer {
	return &refreshTimeoutBuffer{
		entries: make(map[refreshTimeoutKey]*refreshTimeoutBufEntry),
	}
}

func (b *refreshTimeoutBuffer) add(key refreshTimeoutKey, increment time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	e := b.entries[key]
	if e == nil {
		e = &refreshTimeoutBufEntry{}
		b.entries[key] = e
	}

	e.pending += increment
}

func (b *refreshTimeoutBuffer) take(key refreshTimeoutKey) (sum time.Duration, ok bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	e := b.entries[key]
	if e == nil || e.pending <= 0 {
		return 0, false
	}

	sum = e.pending
	e.pending = 0
	return sum, true
}

func (b *refreshTimeoutBuffer) hasPending(key refreshTimeoutKey) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	e := b.entries[key]
	return e != nil && e.pending > 0
}

func (b *refreshTimeoutBuffer) shouldFlush(key refreshTimeoutKey) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	e := b.entries[key]
	if e == nil {
		return true
	}

	b.maybeExpireLocked(key, e)

	e = b.entries[key]
	if e == nil {
		return true
	}

	return e.lastFlushAt.IsZero() || time.Since(e.lastFlushAt) >= refreshTimeoutFlushInterval
}

func (b *refreshTimeoutBuffer) markFlushed(key refreshTimeoutKey, timeoutAt time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()

	e := b.entries[key]
	if e == nil {
		e = &refreshTimeoutBufEntry{}
		b.entries[key] = e
	}

	e.lastFlushAt = time.Now()
	e.lastTimeoutAt = timeoutAt
}

// maybeExpireLocked drops idle entries once they are past the debounce window
// so the map does not grow without bound.
func (b *refreshTimeoutBuffer) maybeExpireLocked(key refreshTimeoutKey, e *refreshTimeoutBufEntry) {
	if e.pending > 0 || e.timer != nil || e.lastFlushAt.IsZero() {
		return
	}

	if time.Since(e.lastFlushAt) >= 2*refreshTimeoutFlushInterval {
		delete(b.entries, key)
	}
}

func (b *refreshTimeoutBuffer) lastTimeout(key refreshTimeoutKey) (time.Time, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	e := b.entries[key]
	if e == nil || e.lastTimeoutAt.IsZero() {
		return time.Time{}, false
	}

	return e.lastTimeoutAt.Add(e.pending), true
}

// scheduleTrailingFlush arms a single timer to flush remaining pending increments
// after the debounce window. flushKey must re-enter the normal flush path.
func (b *refreshTimeoutBuffer) scheduleTrailingFlush(key refreshTimeoutKey, flushKey func(refreshTimeoutKey)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	e := b.entries[key]
	if e == nil || e.pending <= 0 || e.timer != nil {
		return
	}

	e.timer = time.AfterFunc(refreshTimeoutFlushInterval, func() {
		b.mu.Lock()
		if entry := b.entries[key]; entry != nil {
			entry.timer = nil
		}
		b.mu.Unlock()

		flushKey(key)
	})
}

func (b *refreshTimeoutBuffer) stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for key, e := range b.entries {
		if e.timer != nil {
			e.timer.Stop()
			e.timer = nil
		}
		delete(b.entries, key)
	}
}
