package hatchet

import (
	"github.com/hatchet-dev/hatchet/sdks/go/internal"
	"github.com/rs/zerolog"
)

// WorkerOption configures a worker instance.
type WorkerOption func(*workerConfig)

type workerConfig struct {
	workflows    []internal.WorkflowBase
	slots        int
	durableSlots int
	labels       map[string]any
	logger       *zerolog.Logger
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
