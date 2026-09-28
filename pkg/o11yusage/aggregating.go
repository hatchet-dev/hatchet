package o11yusage

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Kind is a separately buffered ingest source. Both kinds flush together but
// stay distinct through to Autumn event names.
type Kind string

const (
	KindLogs Kind = "logs"
	KindOtel Kind = "otel"
)

func (k Kind) valid() bool {
	return k == KindLogs || k == KindOtel
}

// TenantBytes is one tenant's buffered ingest since the last flush.
type TenantBytes struct {
	Logs int64 `json:"logs,omitempty"`
	Otel int64 `json:"otel,omitempty"`
}

// FlushFunc receives a snapshot of tenant → log/otel bytes since the last flush.
// A non-nil error is logged and the snapshot is dropped (best-effort undercount).
type FlushFunc func(tenants map[uuid.UUID]TenantBytes) error

type counterKey struct {
	TenantID uuid.UUID
	Kind     Kind
}

// evictedCount marks a counter that snapshot removed from the map. Add can
// hold a reference to such a counter, loaded just before the removal. An
// increment that lands on it returns a negative number, which tells the
// writer that its bytes were not counted and that it must add them to a
// fresh counter.
const evictedCount = math.MinInt64 / 2

// Aggregator batches per-tenant, per-kind byte counts and flushes them on an
// interval, the same pattern as analytics.Aggregator. Add is the non-blocking
// hot path. A nil FlushFunc disables the ticker.
type Aggregator struct {
	done     chan struct{}
	flushFn  FlushFunc
	l        *zerolog.Logger
	counters sync.Map
	wg       sync.WaitGroup
	interval time.Duration
	flushMu  sync.Mutex
	stopOnce sync.Once
}

// NewAggregator returns an aggregator. If fn is nil, Start is a no-op.
func NewAggregator(l *zerolog.Logger, interval time.Duration, fn FlushFunc) *Aggregator {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	return &Aggregator{
		done:     make(chan struct{}),
		flushFn:  fn,
		l:        l,
		interval: interval,
	}
}

// AddLogs increments the tenant's log-ingest counter. Safe on a nil receiver.
func (a *Aggregator) AddLogs(tenantID uuid.UUID, n int64) {
	a.Add(tenantID, KindLogs, n)
}

// AddOtel increments the tenant's OTLP-ingest counter. Safe on a nil receiver.
func (a *Aggregator) AddOtel(tenantID uuid.UUID, n int64) {
	a.Add(tenantID, KindOtel, n)
}

// Add increments the tenant's counter for kind. Safe on a nil receiver.
func (a *Aggregator) Add(tenantID uuid.UUID, kind Kind, n int64) {
	if a == nil || a.flushFn == nil || n <= 0 || tenantID == uuid.Nil || !kind.valid() {
		return
	}

	key := counterKey{TenantID: tenantID, Kind: kind}

	// Each pass retries only when snapshot evicted the counter between the
	// lookup and the increment, which is rare, so the common case is one pass.
	for {
		if v, ok := a.counters.Load(key); ok && v.(*atomic.Int64).Add(n) > 0 {
			return
		}

		c := &atomic.Int64{}
		c.Add(n)
		existing, loaded := a.counters.LoadOrStore(key, c)
		if !loaded || existing.(*atomic.Int64).Add(n) > 0 {
			return
		}
	}
}

// Start runs the flush ticker. Safe on a nil receiver or when no flush func is set.
func (a *Aggregator) Start() {
	if a == nil || a.flushFn == nil {
		return
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		ticker := time.NewTicker(a.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				a.flush()
			case <-a.done:
				a.flush()
				return
			}
		}
	}()
}

// Shutdown stops the ticker and flushes remaining counters once.
func (a *Aggregator) Shutdown() error {
	if a == nil || a.flushFn == nil {
		return nil
	}

	a.stopOnce.Do(func() {
		close(a.done)
	})
	a.wg.Wait()
	return nil
}

func (a *Aggregator) flush() {
	if a.flushFn == nil {
		return
	}

	if !a.flushMu.TryLock() {
		if a.l != nil {
			a.l.Error().Dur("interval", a.interval).Msg("o11y usage flush still running, skipping interval")
		}
		return
	}
	defer a.flushMu.Unlock()

	tenants := a.snapshot()
	if len(tenants) == 0 {
		return
	}

	if err := a.flushFn(tenants); err != nil && a.l != nil {
		a.l.Error().Err(err).Int("tenants", len(tenants)).Msg("o11y usage flush failed, dropping snapshot")
	}
}

func (a *Aggregator) snapshot() map[uuid.UUID]TenantBytes {
	out := make(map[uuid.UUID]TenantBytes)
	a.counters.Range(func(key, val any) bool {
		ck := key.(counterKey)
		c := val.(*atomic.Int64)
		n := c.Swap(0)
		if n <= 0 {
			a.counters.Delete(key)
			// An Add that loaded this counter before the delete can still
			// increment it. Sealing the counter turns every later increment
			// into a retry against a fresh counter, and the seal returns
			// whatever landed between the swap above and now, which would
			// otherwise be lost with the counter.
			n = c.Swap(evictedCount)
			if n <= 0 {
				return true
			}
		}
		cur := out[ck.TenantID]
		switch ck.Kind {
		case KindLogs:
			cur.Logs += n
		case KindOtel:
			cur.Otel += n
		}
		out[ck.TenantID] = cur
		return true
	})
	return out
}
