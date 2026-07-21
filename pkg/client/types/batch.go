package types

import "time"

// BatchConfig configures batching behavior for a batch task: concurrent task runs are
// buffered and dispatched together as a single execution once MaxSize is reached,
// MaxInterval elapses, or (if GroupKey is set) once GroupMaxRuns concurrent batches per
// group are exceeded.
type BatchConfig struct {
	// MaxSize is the maximum number of items buffered before the batch is flushed.
	// Required, must be positive.
	MaxSize int32

	// MaxInterval is the maximum time to wait before flushing a partially-filled batch.
	// Optional; when unset, batches only flush on MaxSize.
	MaxInterval *time.Duration

	// GroupKey is a CEL expression evaluated against each item's input to partition items
	// into independent batches (e.g. "input.group"). Optional; when unset, all items
	// buffered for the task share a single batch.
	GroupKey *string

	// GroupMaxRuns limits the number of concurrent batches per group. Optional.
	GroupMaxRuns *int32

	// BroadcastOutput, when true, means the handler returns a single output value that is
	// broadcast as the result to every member of the batch, instead of a map keyed by
	// batch member id.
	BroadcastOutput bool
}
