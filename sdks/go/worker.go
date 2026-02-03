package hatchet

import (
	"github.com/rs/zerolog"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/sdks/go/internal"
)

// SlotType represents supported slot types.
type SlotType string

const (
	SlotTypeDefault SlotType = "default"
	SlotTypeDurable SlotType = "durable"
)

// WorkerOption configures a worker instance.
type WorkerOption func(*workerConfig)

type workerConfig struct {
	workflows       []WorkflowBase
	slots           int
	slotsSet        bool
	durableSlots    int
	durableSlotsSet bool
	slotCapacities  map[SlotType]int
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
		config.durableSlotsSet = true
	}
}

// WithSlotCapacities sets slot capacities for this worker (slot_type -> units).
func WithSlotCapacities(slotCapacities map[SlotType]int) WorkerOption {
	return func(config *workerConfig) {
		config.slotCapacities = slotCapacities
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
