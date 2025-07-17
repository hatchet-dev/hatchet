package hatchet

import (
	"github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/rs/zerolog"
)

// WorkerOption configures a worker instance.
type WorkerOption func(*workerConfig)

type workerConfig struct {
	workflows    []workflow.WorkflowBase
	slots        int
	labels       worker.WorkerLabels
	logger       *zerolog.Logger
	logLevel     string
	durableSlots int
}

// WithWorkflows registers workflows with the worker.
func WithWorkflows(workflows ...workflow.WorkflowBase) WorkerOption {
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
func WithLabels(labels worker.WorkerLabels) WorkerOption {
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

// WithLogLevel sets the logging level for the worker.
func WithLogLevel(logLevel string) WorkerOption {
	return func(config *workerConfig) {
		config.logLevel = logLevel
	}
}

// WithDurableSlots sets the maximum number of concurrent durable task runs.
func WithDurableSlots(durableSlots int) WorkerOption {
	return func(config *workerConfig) {
		config.durableSlots = durableSlots
	}
}