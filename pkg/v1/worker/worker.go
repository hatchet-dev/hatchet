// Deprecated: This package is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package worker

import (
	"context"
	"fmt"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/v1/features"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/rs/zerolog"
)

// Deprecated: Worker is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type Worker interface {
	// Start begins worker execution in a non-blocking manner and returns a cleanup function.
	// the cleanup function should be called when the worker needs to be stopped.
	Start() (func() error, error)

	// StartBlocking begins worker execution and blocks until the process is interrupted.
	StartBlocking(ctx context.Context) error

	// RegisterWorkflows registers one or more workflows with the worker.
	RegisterWorkflows(workflows ...workflow.WorkflowBase) error

	// IsPaused checks if the worker is paused
	IsPaused(ctx context.Context) (bool, error)

	// Pause pauses the worker
	Pause(ctx context.Context) error

	// Unpause resumes the paused worker
	Unpause(ctx context.Context) error
}

// Deprecated: WorkerLabels is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type WorkerLabels map[string]interface{}

// Deprecated: WorkerOpts is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type WorkerOpts struct {
	// (required) the friendly name of the worker
	Name string

	// (optional) a list of workflows to register on the worker. If not provided, the worker will not run any workflows.
	Workflows []workflow.WorkflowBase

	// (optional) maximum number of concurrent runs on this worker, defaults to 100
	Slots int

	// (optional) labels to set on the worker
	Labels WorkerLabels

	// (optional) logger to use for the worker
	Logger *zerolog.Logger

	// (optional) log level
	LogLevel string

	// (optional) maximum number of concurrent runs for durable tasks, defaults to 1000
	DurableSlots int
}

// Deprecated: WorkerImpl is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type WorkerImpl struct {
	// v0 is the client used to communicate with the hatchet API.
	v0 v0Client.Client

	// v1 workers client
	workers features.WorkersClient

	// worker is the underlying worker implementation.
	worker *worker.Worker

	// name is the friendly name of the worker.
	name string

	// workflows is a slice of workflows registered with this worker.
	workflows []workflow.WorkflowBase

	// slots is the maximum number of concurrent runs allowed on this worker.
	slots int

	// durableSlots is the maximum number of concurrent durable tasks allowed.
	durableSlots int

	// logLevel is the log level for this worker
	logLevel string

	// logger is the logger used for this worker
	logger *zerolog.Logger

	// labels are the labels assigned to this worker
	labels WorkerLabels
}

// Deprecated: NewWorker is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func NewWorker(workersClient features.WorkersClient, v0 v0Client.Client, opts WorkerOpts) (Worker, error) {
	w := &WorkerImpl{
		v0:        v0,
		workers:   workersClient,
		name:      opts.Name,
		logLevel:  opts.LogLevel,
		logger:    opts.Logger,
		labels:    opts.Labels,
		workflows: opts.Workflows,
	}

	if opts.Slots == 0 {
		w.slots = 100 // default to 100 slots
	} else {
		w.slots = opts.Slots
	}

	if opts.DurableSlots == 0 {
		w.durableSlots = 1000 // default to 1000 durable slots
	} else {
		w.durableSlots = opts.DurableSlots
	}

	// Don't create workers yet - they'll be created on demand when workflows are registered

	// register the workflows
	err := w.RegisterWorkflows(w.workflows...)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// Deprecated: NamedFunction is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type NamedFunction struct {
	ActionID string
	Fn       workflow.WrappedTaskFn
}

// RegisterWorkflows registers one or more workflows with the worker.
// it converts the workflows to the format expected by the underlying worker implementation
// and registers both the workflow definitions and their action functions.
// returns an error if registration fails.
func (w *WorkerImpl) RegisterWorkflows(workflows ...workflow.WorkflowBase) error {
	w.workflows = append(w.workflows, workflows...)

	for _, workflow := range workflows {
		dump, fns, durableFns, onFailureFn := workflow.Dump()

		hasAnyTasks := len(fns) > 0 || len(durableFns) > 0 || (dump.OnFailureTask != nil && onFailureFn != nil)

		// Create worker on demand if needed and not already created
		if hasAnyTasks && w.worker == nil {
			totalRuns := w.slots + w.durableSlots
			opts := []worker.WorkerOpt{
				worker.WithClient(w.v0),
				worker.WithName(w.name),
				worker.WithSlots(totalRuns),
				worker.WithDurableSlots(w.durableSlots),
				worker.WithLogger(w.logger),
				worker.WithLogLevel(w.logLevel),
				worker.WithLabels(w.labels),
			}

			if w.logger != nil {
				opts = append(opts, worker.WithLogger(w.logger))
			}

			wkr, err := worker.NewWorker(
				opts...,
			)
			if err != nil {
				return err
			}
			w.worker = wkr
		}

		// Register workflow with worker if it exists
		if w.worker != nil {
			err := w.worker.RegisterWorkflowV1(dump)
			if err != nil {
				return err
			}

			// Register non-durable actions
			for _, namedFn := range fns {
				err := w.worker.RegisterAction(namedFn.ActionID, namedFn.Fn)
				if err != nil {
					return err
				}
			}

			// Register durable actions on the same worker
			for _, namedFn := range durableFns {
				err := w.worker.RegisterAction(namedFn.ActionID, namedFn.Fn)
				if err != nil {
					return err
				}
			}

			if dump.OnFailureTask != nil && onFailureFn != nil {
				actionId := dump.OnFailureTask.Action
				err := w.worker.RegisterAction(actionId, onFailureFn)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Start begins worker execution in a non-blocking manner.
// returns a cleanup function to be called when the worker should be stopped,
// and any error encountered during startup.
func (w *WorkerImpl) Start() (func() error, error) {
	if w.worker == nil {
		return func() error { return nil }, nil
	}

	cleanup, err := w.worker.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start worker %s: %w", *w.worker.ID(), err)
	}

	return cleanup, nil
}

// StartBlocking begins worker execution and blocks until the process is interrupted.
// this method handles graceful shutdown via interrupt signals.
// returns any error encountered during startup or shutdown.
func (w *WorkerImpl) StartBlocking(ctx context.Context) error {
	cleanup, err := w.Start()
	if err != nil {
		return err
	}

	<-ctx.Done()

	err = cleanup()
	if err != nil {
		return err
	}
	return nil
}

// IsPaused checks if the worker is paused
func (w *WorkerImpl) IsPaused(ctx context.Context) (bool, error) {
	if w.worker == nil {
		return false, nil
	}

	return w.workers.IsPaused(ctx, *w.worker.ID())
}

// Pause pauses the worker
func (w *WorkerImpl) Pause(ctx context.Context) error {
	if w.worker == nil {
		return nil
	}

	_, err := w.workers.Pause(ctx, *w.worker.ID())
	return err
}

// Unpause resumes the paused worker
func (w *WorkerImpl) Unpause(ctx context.Context) error {
	if w.worker == nil {
		return nil
	}

	_, err := w.workers.Unpause(ctx, *w.worker.ID())
	return err
}
