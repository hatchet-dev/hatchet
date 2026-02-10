package eviction

import "time"

// Policy defines task-scoped eviction parameters for durable tasks.
//
// A nil Policy on a durable run means the run is never eligible for eviction.
// TTL applies to time spent in SDK-instrumented "waiting" states (e.g. WaitFor, SleepFor).
type Policy struct {
	// TTL is the maximum continuous waiting duration before the run becomes
	// TTL-eligible for eviction. A nil TTL disables TTL-based eviction.
	TTL *time.Duration

	// AllowCapacityEviction controls whether this task may be evicted when
	// the worker is under durable-slot pressure.
	AllowCapacityEviction bool

	// Priority determines eviction order when multiple candidates exist.
	// Lower values are evicted first.
	Priority int
}

// DefaultPolicy returns sensible defaults for durable task eviction.
func DefaultPolicy() *Policy {
	ttl := 15 * time.Minute
	return &Policy{
		TTL:                   &ttl,
		AllowCapacityEviction: true,
		Priority:              0,
	}
}
