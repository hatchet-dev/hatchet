package analytics

import (
	"hash/fnv"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type counterKey struct {
	Resource  Resource
	Action    Action
	TenantID  uuid.UUID
	TokenID   uuid.UUID
	PropsHash uint64
}

type counterEntry struct {
	count   *atomic.Int64
	props   Properties
	tokenID *uuid.UUID
	// First and last event times since the last flush. Zero means unset.
	firstEventAtNanos atomic.Int64
	lastEventAtNanos  atomic.Int64
}

// The first marker is set once and the last marker only advances
func (e *counterEntry) observe(nanos int64) {
	e.firstEventAtNanos.CompareAndSwap(0, nanos)

	for {
		last := e.lastEventAtNanos.Load()
		if nanos <= last {
			return
		}
		if e.lastEventAtNanos.CompareAndSwap(last, nanos) {
			return
		}
	}
}

// FlushFunc receives one bucket's totals for the interval since the previous
// flush: the number of events counted, and the times of the first and last
// events. The times are zero when unknown and can disagree with the count by
// one event at the interval boundary.
type FlushFunc func(resource Resource, action Action, tenantID uuid.UUID, tokenID *uuid.UUID, count int64, firstEventAt, lastEventAt time.Time, properties Properties)

// Aggregator batches Count calls into periodic flushes. It is intended to be
// embedded inside an Analytics implementation (e.g. PosthogAnalytics) so that
// the implementation's Count method is non-blocking and high-throughput.
type Aggregator struct {
	done     chan struct{}
	flushFn  FlushFunc
	l        *zerolog.Logger
	counters sync.Map
	wg       sync.WaitGroup
	interval time.Duration
	maxKeys  int64
	keyCount atomic.Int64
	flushMu  sync.Mutex
	disabled bool
	now      func() time.Time
}

func NewAggregator(l *zerolog.Logger, enabled bool, interval time.Duration, maxKeys int64, fn FlushFunc) *Aggregator {
	if interval == 0 {
		interval = 60 * time.Minute
	}
	if maxKeys <= 0 {
		maxKeys = 500
	}

	return &Aggregator{
		done:     make(chan struct{}),
		interval: interval,
		flushFn:  fn,
		maxKeys:  maxKeys,
		l:        l,
		disabled: !enabled,
		now:      time.Now,
	}
}

// Count increments the aggregated counter for the given resource/action/tenant/token
// and optional properties. Properties are hashed into the bucket key so that
// different feature combinations (e.g. priority=1 vs priority=3) are counted
// separately. This is the non-blocking hot path: sync.Map.Load + atomic.Add +
// the marker update on the common case.
func (a *Aggregator) Count(resource Resource, action Action, tenantID uuid.UUID, tokenID *uuid.UUID, n int64, props ...Properties) {
	if a.disabled {
		return
	}

	var p Properties
	if len(props) > 0 {
		p = props[0]
	}

	var tid uuid.UUID
	if tokenID != nil {
		tid = *tokenID
	}

	key := counterKey{
		Resource:  resource,
		Action:    action,
		TenantID:  tenantID,
		TokenID:   tid,
		PropsHash: hashProps(p),
	}

	nanos := a.now().UnixNano()

	if v, ok := a.counters.Load(key); ok {
		e := v.(*counterEntry)
		e.count.Add(n)
		e.observe(nanos)
		return
	}

	if a.keyCount.Load() >= a.maxKeys {
		a.l.Error().Int64("max_keys", a.maxKeys).Str("resource", string(resource)).Str("action", string(action)).Str("tenant_id", tenantID.String()).Msg("aggregator at max keys, dropping event")
		return
	}

	c := &atomic.Int64{}
	c.Add(n)
	entry := &counterEntry{count: c, props: p, tokenID: tokenID}
	entry.observe(nanos)
	if existing, loaded := a.counters.LoadOrStore(key, entry); loaded {
		e := existing.(*counterEntry)
		e.count.Add(n)
		e.observe(nanos)
	} else {
		a.keyCount.Add(1)
	}
}

func (a *Aggregator) Start() {
	if a.disabled {
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

func (a *Aggregator) Shutdown() {
	close(a.done)
	a.wg.Wait()
}

func (a *Aggregator) flush() {
	if !a.flushMu.TryLock() {
		a.l.Error().Dur("interval", a.interval).Msg("aggregator flush still running, skipping interval")
		return
	}
	defer a.flushMu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			a.l.Error().Interface("panic", r).Msg("recovered panic in aggregator flush")
		}
	}()

	a.counters.Range(func(key, val any) bool {
		k := key.(counterKey)
		e := val.(*counterEntry)
		count := e.count.Swap(0)
		if count > 0 {
			// The count swap and the marker swaps are separate operations,
			// so an event arriving during this flush can have its count
			// taken by this flush and its time left for the next one, or
			// the reverse. Each flush can misplace at most one event's time
			// in each bucket. A flushed bucket can therefore carry a count
			// with zero markers, and the delete branch below can discard a
			// marker whose count was already flushed. A bucket's first
			// event is never misplaced, because Count fills the markers
			// before publishing a new bucket, so the earliest
			// firstEventAt in a bucket's lifetime is exact. Preventing
			// the misplacement would require Count to take a lock.
			firstNanos := e.firstEventAtNanos.Swap(0)
			lastNanos := e.lastEventAtNanos.Swap(0)

			var firstEventAt, lastEventAt time.Time
			if firstNanos != 0 {
				firstEventAt = time.Unix(0, firstNanos).UTC()
			}
			if lastNanos != 0 {
				lastEventAt = time.Unix(0, lastNanos).UTC()
			}

			a.flushFn(k.Resource, k.Action, k.TenantID, e.tokenID, count, firstEventAt, lastEventAt, e.props)
		} else {
			a.counters.Delete(key)
			a.keyCount.Add(-1)
		}
		return true
	})
}

// hashProps produces a stable FNV-1a hash of sorted key=value pairs.
// A nil/empty map always returns 0 so callers without properties pay no cost.
// Values are restricted to string, bool, and integer types.
func hashProps(m Properties) uint64 {
	if len(m) == 0 {
		return 0
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := fnv.New64a()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{'='})
		switch v := m[k].(type) {
		case string:
			h.Write([]byte(v))
		case bool:
			if v {
				h.Write([]byte{'1'})
			} else {
				h.Write([]byte{'0'})
			}
		case int:
			h.Write(strconv.AppendInt(nil, int64(v), 10))
		case int64:
			h.Write(strconv.AppendInt(nil, v, 10))
		case int32:
			h.Write(strconv.AppendInt(nil, int64(v), 10))
		}
		h.Write([]byte{0})
	}
	return h.Sum64()
}
