package repository

import (
	"sync"
	"time"
)

// TenantRateLimitedError is returned by TenantLimit().CanCreate and TenantControl().Admit for new
// runs and events of a tenant over the ingest rate an operator set. It wraps ErrResourceExhausted.
type TenantRateLimitedError struct {
	// RetryAfter is how long until the same request would be admitted, if nothing else is.
	RetryAfter time.Duration
}

func (e *TenantRateLimitedError) Error() string {
	return ErrResourceExhausted.Error() + ": tenant is rate limited by an operator"
}

func (e *TenantRateLimitedError) Unwrap() error {
	return ErrResourceExhausted
}

// ingestBucket meters one tenant's new runs and events in one process: a token bucket that refills
// at perSecond tokens per second and holds at most one second of them. It keeps the time at which
// it is full again rather than a token count, so the arithmetic is exact: each item costs
// 1/perSecond of a second, and the bucket holds a request's tokens when fullAt is at most one
// second minus the request's cost away.
type ingestBucket struct {
	perSecond int32

	mu sync.Mutex
	// fullAt is when the bucket is full again; a time before now (or zero) means it is full.
	fullAt time.Time
}

func newIngestBucket(perSecond int32) *ingestBucket {
	return &ingestBucket{perSecond: perSecond}
}

// take admits a request for items new runs or events and takes their tokens, or takes nothing and
// returns how long until the request would be admitted. A request is admitted whole when the bucket
// holds that many tokens; a request for more than the bucket can hold is admitted when it is full
// and leaves it in debt, so a large batch is slowed, not starved.
func (b *ingestBucket) take(items int, now time.Time) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	cost := time.Duration(int64(items) * int64(time.Second) / int64(b.perSecond))
	start := b.fullAt
	if start.Before(now) {
		start = now
	}

	if wait := start.Add(min(cost, time.Second) - time.Second).Sub(now); wait > 0 {
		return wait
	}

	b.fullAt = start.Add(cost)

	return 0
}
