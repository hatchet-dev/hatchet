package hatchet

import "time"

// EvictionPolicy configures how durable tasks are evicted from worker slots
// when they are in a waiting state (e.g. sleeping, waiting for events, waiting for children).
type EvictionPolicy struct {
	// TTL is the maximum continuous waiting duration before TTL-eligible eviction.
	// A zero value means no TTL-based eviction.
	TTL time.Duration

	// AllowCapacityEviction controls whether this task may be evicted under durable-slot pressure.
	AllowCapacityEviction bool

	// Priority determines eviction order when multiple candidates exist.
	// Lower values are evicted first.
	Priority int
}

// DefaultDurableTaskEvictionPolicy provides sensible defaults for durable task eviction.
var DefaultDurableTaskEvictionPolicy = &EvictionPolicy{
	TTL:                   15 * time.Minute,
	AllowCapacityEviction: true,
	Priority:              0,
}

// WithEvictionPolicy sets the eviction policy for a durable task.
func WithEvictionPolicy(policy *EvictionPolicy) TaskOption {
	return func(config *taskConfig) {
		config.evictionPolicy = policy
	}
}
