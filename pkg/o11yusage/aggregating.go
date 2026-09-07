package o11yusage

import (
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
	if v, ok := a.counters.Load(key); ok {
		v.(*atomic.Int64).Add(n)
		return
	}

	c := &atomic.Int64{}
	c.Add(n)
	if existing, loaded := a.counters.LoadOrStore(key, c); loaded {
		existing.(*atomic.Int64).Add(n)
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
		n := val.(*atomic.Int64).Swap(0)
		if n <= 0 {
			a.counters.Delete(key)
			return true
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
