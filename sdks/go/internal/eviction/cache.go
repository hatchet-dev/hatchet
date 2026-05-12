package eviction

import (
	"fmt"
	"sync"
	"time"
)

type EvictionCause string

const (
	EvictionCauseTTLExceeded      EvictionCause = "ttl_exceeded"
	EvictionCauseCapacityPressure EvictionCause = "capacity_pressure"
	EvictionCauseWorkerShutdown   EvictionCause = "worker_shutdown"
)

// EvictionPolicy mirrors the SDK-level EvictionPolicy for internal use.
type EvictionPolicy struct {
	TTL                   time.Duration
	AllowCapacityEviction bool
	Priority              int
}

// DurableRunRecord tracks the state of a durable run for eviction decisions.
type DurableRunRecord struct {
	RegisteredAt    time.Time
	EvictionPolicy  *EvictionPolicy
	WaitingSince    *time.Time
	Key             string
	StepRunID       string
	WaitKind        string
	WaitResourceID  string
	EvictionReason  string
	InvocationCount int
	waitCount       int
	mu              sync.Mutex
}

// IsWaiting returns true if the run is currently in a waiting state.
func (r *DurableRunRecord) IsWaiting() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.waitCount > 0
}

// WaitCount returns the current wait reference count.
func (r *DurableRunRecord) WaitCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.waitCount
}

// GetWaitingSince returns a copy of the waiting since timestamp.
func (r *DurableRunRecord) GetWaitingSince() *time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.WaitingSince == nil {
		return nil
	}
	t := *r.WaitingSince
	return &t
}

// DurableEvictionCache manages durable run state for eviction decisions.
// Thread-safe via internal mutex.
type DurableEvictionCache struct {
	runs map[string]*DurableRunRecord
	mu   sync.RWMutex
}

// NewDurableEvictionCache creates a new empty cache.
func NewDurableEvictionCache() *DurableEvictionCache {
	return &DurableEvictionCache{
		runs: make(map[string]*DurableRunRecord),
	}
}

// RegisterRun adds a run to the cache.
func (c *DurableEvictionCache) RegisterRun(
	key, stepRunID string,
	invocationCount int,
	now time.Time,
	policy *EvictionPolicy,
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.runs[key] = &DurableRunRecord{
		Key:             key,
		StepRunID:       stepRunID,
		InvocationCount: invocationCount,
		EvictionPolicy:  policy,
		RegisteredAt:    now,
	}
}

// UnregisterRun removes a run from the cache.
func (c *DurableEvictionCache) UnregisterRun(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.runs, key)
}

// Get returns the run record for a given key, or nil if not found.
func (c *DurableEvictionCache) Get(key string) *DurableRunRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.runs[key]
}

// GetAllWaiting returns all run records currently in a waiting state.
func (c *DurableEvictionCache) GetAllWaiting() []*DurableRunRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var result []*DurableRunRecord
	for _, r := range c.runs {
		if r.IsWaiting() {
			result = append(result, r)
		}
	}
	return result
}

// FindKeyByStepRunID looks up a run key by its step run ID (linear scan).
func (c *DurableEvictionCache) FindKeyByStepRunID(stepRunID string) *string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for key, rec := range c.runs {
		if rec.StepRunID == stepRunID {
			k := key
			return &k
		}
	}
	return nil
}

// MarkWaiting increments the wait reference count for a run.
func (c *DurableEvictionCache) MarkWaiting(key string, now time.Time, waitKind, resourceID string) {
	c.mu.RLock()
	rec := c.runs[key]
	c.mu.RUnlock()
	if rec == nil {
		return
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.waitCount++
	if rec.waitCount == 1 {
		rec.WaitingSince = &now
	}
	rec.WaitKind = waitKind
	rec.WaitResourceID = resourceID
}

// MarkActive decrements the wait reference count for a run.
func (c *DurableEvictionCache) MarkActive(key string, now time.Time) {
	c.mu.RLock()
	rec := c.runs[key]
	c.mu.RUnlock()
	if rec == nil {
		return
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	rec.waitCount--
	if rec.waitCount < 0 {
		rec.waitCount = 0
	}
	if rec.waitCount == 0 {
		rec.WaitingSince = nil
		rec.WaitKind = ""
		rec.WaitResourceID = ""
	}
}

// SelectEvictionCandidate selects the best eviction candidate.
// Returns the key of the selected run, or empty string if none.
func (c *DurableEvictionCache) SelectEvictionCandidate(
	now time.Time,
	durableSlots int,
	reserveSlots int,
	minWaitForCapacityEviction time.Duration,
) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var waiting []*DurableRunRecord
	for _, r := range c.runs {
		if r.IsWaiting() && r.EvictionPolicy != nil {
			waiting = append(waiting, r)
		}
	}
	if len(waiting) == 0 {
		return ""
	}

	// Prefer TTL-eligible candidates first
	var ttlEligible []*DurableRunRecord
	for _, r := range waiting {
		r.mu.Lock()
		policy := r.EvictionPolicy
		ws := r.WaitingSince
		r.mu.Unlock()

		if policy != nil && policy.TTL > 0 && ws != nil {
			if now.Sub(*ws) >= policy.TTL {
				ttlEligible = append(ttlEligible, r)
			}
		}
	}

	if len(ttlEligible) > 0 {
		chosen := selectByPriorityAndAge(ttlEligible, now)
		chosen.mu.Lock()
		ttl := time.Duration(0)
		if chosen.EvictionPolicy != nil {
			ttl = chosen.EvictionPolicy.TTL
		}
		chosen.EvictionReason = buildEvictionReason(EvictionCauseTTLExceeded, chosen, ttl)
		chosen.mu.Unlock()
		return chosen.Key
	}

	// Capacity eviction
	if !capacityPressure(durableSlots, reserveSlots, len(waiting)) {
		return ""
	}

	var capacityCandidates []*DurableRunRecord
	for _, r := range waiting {
		r.mu.Lock()
		policy := r.EvictionPolicy
		ws := r.WaitingSince
		r.mu.Unlock()

		if policy != nil && policy.AllowCapacityEviction && ws != nil {
			if now.Sub(*ws) >= minWaitForCapacityEviction {
				capacityCandidates = append(capacityCandidates, r)
			}
		}
	}

	if len(capacityCandidates) == 0 {
		return ""
	}

	chosen := selectByPriorityAndAge(capacityCandidates, now)
	chosen.mu.Lock()
	chosen.EvictionReason = buildEvictionReason(EvictionCauseCapacityPressure, chosen, 0)
	chosen.mu.Unlock()
	return chosen.Key
}

func capacityPressure(durableSlots, reserveSlots, waitingCount int) bool {
	if durableSlots <= 0 {
		return false
	}
	maxWaiting := durableSlots - reserveSlots
	if maxWaiting <= 0 {
		return false
	}
	return waitingCount >= maxWaiting
}

// selectByPriorityAndAge picks the candidate with lowest priority (evict first),
// then oldest waiting_since.
func selectByPriorityAndAge(candidates []*DurableRunRecord, now time.Time) *DurableRunRecord {
	best := candidates[0]
	for _, c := range candidates[1:] {
		c.mu.Lock()
		cPrio := 0
		if c.EvictionPolicy != nil {
			cPrio = c.EvictionPolicy.Priority
		}
		cWS := c.WaitingSince
		c.mu.Unlock()

		best.mu.Lock()
		bPrio := 0
		if best.EvictionPolicy != nil {
			bPrio = best.EvictionPolicy.Priority
		}
		bWS := best.WaitingSince
		best.mu.Unlock()

		if cPrio < bPrio {
			best = c
		} else if cPrio == bPrio {
			if cWS != nil && bWS != nil && cWS.Before(*bWS) {
				best = c
			}
		}
	}
	return best
}

func buildEvictionReason(cause EvictionCause, rec *DurableRunRecord, ttl time.Duration) string {
	waitDesc := rec.WaitKind
	if waitDesc == "" {
		waitDesc = "unknown"
	}
	if rec.WaitResourceID != "" {
		waitDesc = fmt.Sprintf("%s(%s)", waitDesc, rec.WaitResourceID)
	}

	switch cause {
	case EvictionCauseTTLExceeded:
		ttlStr := ""
		if ttl > 0 {
			ttlStr = fmt.Sprintf(" (%s)", ttl)
		}
		return fmt.Sprintf("Wait TTL%s exceeded while waiting on %s", ttlStr, waitDesc)
	case EvictionCauseCapacityPressure:
		return fmt.Sprintf("Worker at capacity while waiting on %s", waitDesc)
	case EvictionCauseWorkerShutdown:
		return fmt.Sprintf("Worker shutdown while waiting on %s", waitDesc)
	default:
		return fmt.Sprintf("Unknown eviction cause while waiting on %s", waitDesc)
	}
}
