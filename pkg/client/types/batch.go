package types

import "time"

// BatchConfig configures batching behavior for a batch task: concurrent task runs are
// buffered and dispatched together as a single execution once MaxSize is reached or
// MaxInterval elapses. While a group has GroupMaxRuns batches running, its ready batches
// stay buffered.
type BatchConfig struct {
	// MaxSize is the maximum number of items buffered before the batch is flushed.
	// Required, must be positive.
	MaxSize int32

	// MaxInterval is the maximum time to wait before flushing a partially-filled batch,
	// measured from when its first item was buffered. Optional; defaults to 1s.
	MaxInterval *time.Duration

	// GroupKey is a CEL expression evaluated against each item's input to partition items
	// into independent batches (e.g. "input.group"). Optional; when unset, all items
	// buffered for the task share a single batch.
	GroupKey *string

	// GroupMaxRuns limits how many batches per group can run at once. Optional; defaults to
	// 100. Without GroupKey, all runs share one group.
	GroupMaxRuns *int32

	// BroadcastOutput, when true, means the handler returns a single output value that is
	// broadcast as the result to every member of the batch, instead of a map keyed by
	// batch member id.
	BroadcastOutput bool
}
