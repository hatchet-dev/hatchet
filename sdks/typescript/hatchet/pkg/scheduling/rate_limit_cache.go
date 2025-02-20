package scheduling

import (
	"fmt"
	"sync"
	"time"
)

// ExhaustedRateLimitCache is a cache of rate limits to their next refill time, which avoids querying queues
// where we know we're already rate-limited.
type ExhaustedRateLimitCache struct {
	rlKeysToRefillTimes sync.Map
	maxCacheDuration    time.Duration
}

// NewExhaustedRateLimitCache creates a new ExhaustedRateLimitCache.
func NewExhaustedRateLimitCache(maxCacheDuration time.Duration) *ExhaustedRateLimitCache {
	return &ExhaustedRateLimitCache{
		maxCacheDuration: maxCacheDuration,
	}
}

type cacheEntry struct {
	minRefillTime time.Time
}

func (rlc *ExhaustedRateLimitCache) Set(tenantId, queue string, exhaustedRateLimitRefillTimes []time.Time) {
	minRefillTime := time.Now().Add(rlc.maxCacheDuration)

	for _, refillTime := range exhaustedRateLimitRefillTimes {
		if refillTime.Before(minRefillTime) {
			minRefillTime = refillTime
		}
	}

	rlc.rlKeysToRefillTimes.Store(getKeyName(tenantId, queue), cacheEntry{
		minRefillTime: minRefillTime,
	})
}

// Get returns true if the rate limit is not exhausted, false otherwise.
func (rlc *ExhaustedRateLimitCache) IsExhausted(tenantId, queue string) bool {
	refillTime, ok := rlc.rlKeysToRefillTimes.Load(getKeyName(tenantId, queue))

	if !ok {
		return false
	}

	isExhausted := refillTime.(cacheEntry).minRefillTime.After(time.Now())

	// if the cache entry is expired, remove it
	if !isExhausted {
		rlc.rlKeysToRefillTimes.Delete(getKeyName(tenantId, queue))
	}

	return isExhausted
}

func getKeyName(tenantId, key string) string {
	return fmt.Sprintf("%s:%s", tenantId, key)
}
