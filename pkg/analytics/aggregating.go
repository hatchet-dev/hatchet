package analytics

import (
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
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
	props   map[string]interface{}
	tokenID *uuid.UUID
}

type FlushFunc func(resource Resource, action Action, tenantID uuid.UUID, tokenID *uuid.UUID, count int64, properties map[string]interface{})

// Aggregator batches Count calls into periodic flushes. It is intended to be
// embedded inside an Analytics implementation (e.g. PosthogAnalytics) so that
// the implementation's Count method is non-blocking and high-throughput.
type Aggregator struct {
	done     chan struct{}
	flushFn  FlushFunc
	counters sync.Map
	wg       sync.WaitGroup
	interval time.Duration
}

func NewAggregator(interval time.Duration, fn FlushFunc) *Aggregator {
	return &Aggregator{
		done:     make(chan struct{}),
		interval: interval,
		flushFn:  fn,
	}
}

// Count increments the aggregated counter for the given resource/action/tenant/token
// and optional properties. Properties are hashed into the bucket key so that
// different feature combinations (e.g. priority=1 vs priority=3) are counted
// separately. This is the non-blocking hot path: sync.Map.Load + atomic.Add
// on the common case.
func (a *Aggregator) Count(resource Resource, action Action, tenantID uuid.UUID, tokenID *uuid.UUID, n int64, props ...map[string]interface{}) {
	var p map[string]interface{}
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

	c := &atomic.Int64{}
	c.Add(n)
	entry := &counterEntry{count: c, props: p, tokenID: tokenID}
	if existing, loaded := a.counters.LoadOrStore(key, entry); loaded {
		existing.(*counterEntry).count.Add(n)
	}
}

func (a *Aggregator) Start() {
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
	a.counters.Range(func(key, val any) bool {
		k := key.(counterKey)
		e := val.(*counterEntry)
		count := e.count.Swap(0)
		if count > 0 {
			a.flushFn(k.Resource, k.Action, k.TenantID, e.tokenID, count, e.props)
		} else {
			a.counters.Delete(key)
		}
		return true
	})
}

// hashProps produces a stable FNV-1a hash of sorted key=value pairs.
// A nil/empty map always returns 0 so callers without properties pay no cost.
func hashProps(m map[string]interface{}) uint64 {
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
		fmt.Fprintf(h, "%s=%v\x00", k, m[k])
	}
	return h.Sum64()
}
