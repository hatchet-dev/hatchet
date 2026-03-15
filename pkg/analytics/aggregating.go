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
}

type FlushFunc func(resource Resource, action Action, tenantID uuid.UUID, tokenID *uuid.UUID, count int64, properties Properties)

// Aggregator batches Count calls into periodic flushes. It is intended to be
// embedded inside an Analytics implementation (e.g. PosthogAnalytics) so that
// the implementation's Count method is non-blocking and high-throughput.
type Aggregator struct {
	done     chan struct{}
	flushFn  FlushFunc
	counters sync.Map
	wg       sync.WaitGroup
	interval time.Duration
	maxKeys  int64
	keyCount atomic.Int64
	l        *zerolog.Logger
	disabled bool
	flushMu  sync.Mutex
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
	}
}

// Count increments the aggregated counter for the given resource/action/tenant/token
// and optional properties. Properties are hashed into the bucket key so that
// different feature combinations (e.g. priority=1 vs priority=3) are counted
// separately. This is the non-blocking hot path: sync.Map.Load + atomic.Add
// on the common case.
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

	if v, ok := a.counters.Load(key); ok {
		v.(*counterEntry).count.Add(n)
		return
	}

	if a.keyCount.Load() >= a.maxKeys {
		a.l.Error().Int64("max_keys", a.maxKeys).Str("resource", string(resource)).Str("action", string(action)).Str("tenant_id", tenantID.String()).Msg("aggregator at max keys, dropping event")
		return
	}

	c := &atomic.Int64{}
	c.Add(n)
	entry := &counterEntry{count: c, props: p, tokenID: tokenID}
	if existing, loaded := a.counters.LoadOrStore(key, entry); loaded {
		existing.(*counterEntry).count.Add(n)
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
			a.flushFn(k.Resource, k.Action, k.TenantID, e.tokenID, count, e.props)
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
