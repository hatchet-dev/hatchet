package hatchet

import (
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/sdks/go/internal"
)

// WorkerOption configures a worker instance.
type WorkerOption func(*workerConfig)

type workerConfig struct {
	workflows    []internal.WorkflowBase
	slots        int
	durableSlots int
	labels       map[string]any
	logger       *zerolog.Logger
	panicHandler func(ctx Context, recovered any)
}

// WithWorkflows registers workflows and standalone tasks with the worker.
// Both workflows and standalone tasks implement the WorkflowBase interface.
func WithWorkflows(workflows ...internal.WorkflowBase) WorkerOption {
	return func(config *workerConfig) {
		config.workflows = workflows
	}
}

// WithSlots sets the maximum number of concurrent workflow runs.
func WithSlots(slots int) WorkerOption {
	return func(config *workerConfig) {
		config.slots = slots
	}
}

// WithLabels assigns labels to the worker for task routing.
func WithLabels(labels map[string]any) WorkerOption {
	return func(config *workerConfig) {
		config.labels = labels
	}
}

// WithLogger sets a custom logger for the worker.
func WithLogger(logger *zerolog.Logger) WorkerOption {
	return func(config *workerConfig) {
		config.logger = logger
	}
}

// WithDurableSlots sets the maximum number of concurrent durable task runs.
func WithDurableSlots(durableSlots int) WorkerOption {
	return func(config *workerConfig) {
		config.durableSlots = durableSlots
	}
}

// WithPanicHandler sets a custom panic handler for the worker.
//
// recovered is the non-nil value that was obtained after calling recover()
func WithPanicHandler(panicHandler func(ctx Context, recovered any)) WorkerOption {
	return func(config *workerConfig) {
		config.panicHandler = panicHandler
	}
}
