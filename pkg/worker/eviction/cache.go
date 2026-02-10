package eviction

import (
	"context"
	"sort"
	"sync"
	"time"
)

// DurableRunRecord tracks the state of a single durable task run for eviction purposes.
type DurableRunRecord struct {
	Key          string
	StepRunId    string
	Ctx          context.Context         // used to check if already cancelled
	Cancel       context.CancelCauseFunc // used to cancel with a cause (e.g. ErrEvicted)
	Eviction     *Policy                 // nil = never evictable
	RegisteredAt time.Time

	// Waiting state
	WaitingSince   *time.Time
	WaitKind       string
	WaitResourceID string
}

// IsWaiting returns true if the run is currently in a waiting state.
func (r *DurableRunRecord) IsWaiting() bool {
	return r.WaitingSince != nil
}

// IsCancelled returns true if the run's context has been cancelled.
func (r *DurableRunRecord) IsCancelled() bool {
	return r.Ctx.Err() != nil
}

// RegisterRunOpts holds the parameters for registering a durable run.
type RegisterRunOpts struct {
	Key       string
	StepRunId string
	Ctx       context.Context
	Cancel    context.CancelCauseFunc
	Now       time.Time
	Eviction  *Policy
}

// DurableEvictionCache defines the interface for tracking durable run state.
type DurableEvictionCache interface {
	RegisterRun(opts RegisterRunOpts)
	UnregisterRun(key string)
	MarkWaiting(key string, now time.Time, waitKind, resourceID string)
	MarkActive(key string)
	SelectEvictionCandidate(now time.Time, durableSlots, reserveSlots int, minWaitForCapacityEviction time.Duration) string
	Get(key string) *DurableRunRecord
}

// InMemoryDurableEvictionCache is a thread-safe in-memory implementation of DurableEvictionCache.
type InMemoryDurableEvictionCache struct {
	runs map[string]*DurableRunRecord
	mu   sync.RWMutex
}

// NewInMemoryDurableEvictionCache creates a new in-memory eviction cache.
func NewInMemoryDurableEvictionCache() *InMemoryDurableEvictionCache {
	return &InMemoryDurableEvictionCache{
		runs: make(map[string]*DurableRunRecord),
	}
}

func (c *InMemoryDurableEvictionCache) RegisterRun(key, stepRunId string, ctx context.Context, cancel context.CancelCauseFunc, now time.Time, eviction *Policy) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.runs[key] = &DurableRunRecord{
		Key:          key,
		StepRunId:    stepRunId,
		Ctx:          ctx,
		Cancel:       cancel,
		Eviction:     eviction,
		RegisteredAt: now,
	}
}

func (c *InMemoryDurableEvictionCache) UnregisterRun(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.runs, key)
}

func (c *InMemoryDurableEvictionCache) Get(key string) *DurableRunRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.runs[key]
}

func (c *InMemoryDurableEvictionCache) MarkWaiting(key string, now time.Time, waitKind, resourceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	rec, ok := c.runs[key]
	if !ok {
		return
	}
	if rec.IsCancelled() {
		return
	}
	rec.WaitingSince = &now
	rec.WaitKind = waitKind
	rec.WaitResourceID = resourceID
}

func (c *InMemoryDurableEvictionCache) MarkActive(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	rec, ok := c.runs[key]
	if !ok {
		return
	}
	rec.WaitingSince = nil
	rec.WaitKind = ""
	rec.WaitResourceID = ""
}

func (c *InMemoryDurableEvictionCache) capacityPressure(durableSlots, reserveSlots, waitingCount int) bool {
	if durableSlots <= 0 {
		return false
	}
	maxWaiting := durableSlots - reserveSlots
	if maxWaiting <= 0 {
		return false
	}
	return waitingCount >= maxWaiting
}

// SelectEvictionCandidate selects a run to evict based on TTL or capacity pressure.
// Returns the key of the selected candidate, or "" if no candidate is eligible.
func (c *InMemoryDurableEvictionCache) SelectEvictionCandidate(
	now time.Time,
	durableSlots, reserveSlots int,
	minWaitForCapacityEviction time.Duration,
) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Collect waiting runs that are eligible for eviction (have an eviction policy).
	var waiting []*DurableRunRecord
	for _, r := range c.runs {
		if r.IsWaiting() && !r.IsCancelled() && r.Eviction != nil {
			waiting = append(waiting, r)
		}
	}

	if len(waiting) == 0 {
		return ""
	}

	// Prefer TTL-eligible candidates first.
	var ttlEligible []*DurableRunRecord
	for _, r := range waiting {
		if r.Eviction.TTL == nil || r.WaitingSince == nil {
			continue
		}
		if now.Sub(*r.WaitingSince) >= *r.Eviction.TTL {
			ttlEligible = append(ttlEligible, r)
		}
	}

	if len(ttlEligible) > 0 {
		sort.Slice(ttlEligible, func(i, j int) bool {
			if ttlEligible[i].Eviction.Priority != ttlEligible[j].Eviction.Priority {
				return ttlEligible[i].Eviction.Priority < ttlEligible[j].Eviction.Priority
			}
			return ttlEligible[i].WaitingSince.Before(*ttlEligible[j].WaitingSince)
		})
		return ttlEligible[0].Key
	}

	// Capacity eviction: only if under pressure and run allows it.
	if !c.capacityPressure(durableSlots, reserveSlots, len(waiting)) {
		return ""
	}

	var capacityCandidates []*DurableRunRecord
	for _, r := range waiting {
		if r.Eviction == nil || !r.Eviction.AllowCapacityEviction {
			continue
		}
		if r.WaitingSince == nil {
			continue
		}
		if now.Sub(*r.WaitingSince) < minWaitForCapacityEviction {
			continue
		}
		capacityCandidates = append(capacityCandidates, r)
	}

	if len(capacityCandidates) == 0 {
		return ""
	}

	sort.Slice(capacityCandidates, func(i, j int) bool {
		if capacityCandidates[i].Eviction.Priority != capacityCandidates[j].Eviction.Priority {
			return capacityCandidates[i].Eviction.Priority < capacityCandidates[j].Eviction.Priority
		}
		return capacityCandidates[i].WaitingSince.Before(*capacityCandidates[j].WaitingSince)
	})

	return capacityCandidates[0].Key
}
