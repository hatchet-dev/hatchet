package hatchet

import (
	"github.com/rs/zerolog"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/sdks/go/internal"
)

// WorkerOption configures a worker instance.
type WorkerOption func(*workerConfig)

type workerConfig struct {
	workflows       []WorkflowBase
	slots           int
	slotsSet        bool
	durableSlots    int
	durableSlotsSet bool
	labels          map[string]any
	logger          *zerolog.Logger
	panicHandler    func(ctx Context, recovered any)
}

type WorkflowBase interface {
	GetName() string
	OnFailure(fn any)

	// Internal use only. Will be removed in the future.
	Dump() (*v1.CreateWorkflowVersionRequest, []internal.NamedFunction, []internal.NamedFunction, internal.WrappedTaskFn)
}

// WithWorkflows registers workflows and standalone tasks with the worker.
// Both workflows and standalone tasks implement the WorkflowBase interface.
func WithWorkflows(workflows ...WorkflowBase) WorkerOption {
	return func(config *workerConfig) {
		config.workflows = workflows
	}
}

// WithSlots sets the maximum number of concurrent workflow runs.
func WithSlots(slots int) WorkerOption {
	return func(config *workerConfig) {
		config.slots = slots
		config.slotsSet = true
	}
}

// WithLabels assigns labels to the worker for task routing.
func WithLabels(labels map[string]any) WorkerOption {
	return func(config *workerConfig) {
		config.labels = labels
	}
}

// WithLogger sets a custom logger for the worker. When not set, the worker inherits the client's logger.
func WithLogger(logger *zerolog.Logger) WorkerOption {
	return func(config *workerConfig) {
		config.logger = logger
	}
}

// resolveWorkerLogger picks the logger a worker should use: an explicitly
// configured worker logger (via WithLogger) always wins; otherwise the client's
// logger is inherited as a sub-logger tagged with service=worker, so worker
// output follows the client's configured level and format. Returns nil when
// neither logger is available.
//
// The client logger already carries its own service field, so the emitted JSON
// repeats the key with the worker value last; JSON consumers and zerolog's
// console writer both resolve to service=worker.
func resolveWorkerLogger(configured *zerolog.Logger, clientLogger *zerolog.Logger) *zerolog.Logger {
	if configured != nil {
		return configured
	}

	if clientLogger == nil {
		return nil
	}

	derived := clientLogger.With().Str("service", "worker").Logger()

	return &derived
}

// WithDurableSlots sets the maximum number of concurrent durable task runs.
func WithDurableSlots(durableSlots int) WorkerOption {
	return func(config *workerConfig) {
		config.durableSlots = durableSlots
		config.durableSlotsSet = true
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
