// package worker provides functionality for creating and managing hatchet workers.
// workers are responsible for executing workflow tasks and communicating with the hatchet API.
package worker

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/v1/features"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// Worker defines the interface for interacting with a hatchet worker.
type Worker interface {
	// Start begins worker execution in a non-blocking manner and returns a cleanup function.
	// the cleanup function should be called when the worker needs to be stopped.
	Start() (func() error, error)

	// StartBlocking begins worker execution and blocks until the process is interrupted.
	StartBlocking(ctx context.Context) error

	// RegisterWorkflows registers one or more workflows with the worker.
	RegisterWorkflows(workflows ...workflow.WorkflowBase) error

	// IsPaused checks if all worker instances are paused
	IsPaused(ctx context.Context) (bool, error)

	// Pause pauses all worker instances
	Pause(ctx context.Context) error

	// Unpause resumes all paused worker instances
	Unpause(ctx context.Context) error
}

// WorkerLabels represents a map of labels that can be assigned to a worker
// for filtering and identification purposes.
type WorkerLabels map[string]interface{}

// CreateOpts defines the options for creating a new worker.
type WorkerOpts struct {
	Labels       WorkerLabels
	Logger       *zerolog.Logger
	Name         string
	LogLevel     string
	Workflows    []workflow.WorkflowBase
	Slots        int
	DurableSlots int
}

// WorkerImpl is the concrete implementation of the Worker interface.
type WorkerImpl struct {
	v0               v0Client.Client
	workers          features.WorkersClient
	nonDurableWorker *worker.Worker
	durableWorker    *worker.Worker
	logger           *zerolog.Logger
	labels           WorkerLabels
	name             string
	logLevel         string
	workflows        []workflow.WorkflowBase
	slots            int
	durableSlots     int
}

// NewWorker creates and configures a new Worker with the provided client and options.
// additional functional options can be provided to further customize the worker configuration.
// returns the created Worker interface and any error encountered during creation.
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

// NamedFunction represents a function with its associated action ID
type NamedFunction struct {
	Fn       workflow.WrappedTaskFn
	ActionID string
}

// RegisterWorkflows registers one or more workflows with the worker.
// it converts the workflows to the format expected by the underlying worker implementation
// and registers both the workflow definitions and their action functions.
// returns an error if registration fails.
func (w *WorkerImpl) RegisterWorkflows(workflows ...workflow.WorkflowBase) error {
	w.workflows = append(w.workflows, workflows...)

	for _, workflow := range workflows {
		dump, fns, durableFns, onFailureFn := workflow.Dump()

		// Check if there are non-durable tasks in this workflow
		hasNonDurableTasks := len(fns) > 0 || (dump.OnFailureTask != nil && onFailureFn != nil)
		hasDurableTasks := len(durableFns) > 0

		// Create non-durable worker on demand if needed and not already created
		if hasNonDurableTasks && w.nonDurableWorker == nil {
			opts := []worker.WorkerOpt{
				worker.WithClient(w.v0),
				worker.WithName(w.name),
				worker.WithMaxRuns(w.slots),
				worker.WithLogger(w.logger),
				worker.WithLogLevel(w.logLevel),
				worker.WithLabels(w.labels),
			}

			if w.logger != nil {
				opts = append(opts, worker.WithLogger(w.logger))
			}

			nonDurableWorker, err := worker.NewWorker(
				opts...,
			)
			if err != nil {
				return err
			}
			w.nonDurableWorker = nonDurableWorker
		}

		// Create durable worker on demand if needed and not already created
		if hasDurableTasks && w.durableWorker == nil {
			// Reuse logger from main worker if exists
			var logger *zerolog.Logger
			if w.nonDurableWorker != nil {
				logger = w.nonDurableWorker.Logger()
			}

			labels := make(map[string]interface{})
			for k, v := range w.labels {
				labels[k] = fmt.Sprintf("%v-durable", v)
			}

			opts := []worker.WorkerOpt{
				worker.WithClient(w.v0),
				worker.WithName(w.name + "-durable"),
				worker.WithMaxRuns(w.durableSlots),
				worker.WithLogger(logger),
				worker.WithLogLevel(w.logLevel),
				worker.WithLabels(labels),
			}

			durableWorker, err := worker.NewWorker(
				opts...,
			)
			if err != nil {
				return err
			}
			w.durableWorker = durableWorker
		}

		// Register workflow with non-durable worker if it exists
		if w.nonDurableWorker != nil {
			err := w.nonDurableWorker.RegisterWorkflowV1(dump)
			if err != nil {
				return err
			}

			// Register non-durable actions
			for _, namedFn := range fns {
				err := w.nonDurableWorker.RegisterAction(namedFn.ActionID, namedFn.Fn)
				if err != nil {
					return err
				}
			}

			if dump.OnFailureTask != nil && onFailureFn != nil {
				actionId := dump.OnFailureTask.Action
				err := w.nonDurableWorker.RegisterAction(actionId, onFailureFn)
				if err != nil {
					return err
				}
			}
		}

		// Register durable actions with durable worker
		if w.durableWorker != nil {
			err := w.durableWorker.RegisterWorkflowV1(dump)
			if err != nil {
				return err
			}

			for _, namedFn := range durableFns {
				err := w.durableWorker.RegisterAction(namedFn.ActionID, namedFn.Fn)
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
	// Create slice of workers that exist
	var workers []*worker.Worker
	if w.nonDurableWorker != nil {
		workers = append(workers, w.nonDurableWorker)
	}
	if w.durableWorker != nil {
		workers = append(workers, w.durableWorker)
	}

	// Track cleanup functions with a mutex to safely access from multiple goroutines
	var cleanupFuncs []func() error
	var cleanupMu sync.Mutex

	// Use errgroup to start workers concurrently
	g := new(errgroup.Group)

	// Start all workers concurrently
	for i := range workers {
		worker := workers[i] // Capture the worker for the goroutine
		g.Go(func() error {
			cleanup, err := worker.Start()
			if err != nil {
				return fmt.Errorf("failed to start worker %s: %w", *worker.ID(), err)
			}

			cleanupMu.Lock()
			cleanupFuncs = append(cleanupFuncs, cleanup)
			cleanupMu.Unlock()
			return nil
		})
	}

	// Wait for all workers to start
	if err := g.Wait(); err != nil {
		// Clean up any workers that did start
		for _, cleanupFn := range cleanupFuncs {
			_ = cleanupFn()
		}
		return nil, err
	}

	// Return a combined cleanup function that also uses errgroup for concurrent cleanup
	return func() error {
		g := new(errgroup.Group)

		for _, cleanup := range cleanupFuncs {
			cleanupFn := cleanup // Capture the cleanup function for the goroutine
			g.Go(func() error {
				return cleanupFn()
			})
		}

		// Wait for all cleanup operations to complete and return any error
		if err := g.Wait(); err != nil {
			return fmt.Errorf("worker cleanup error: %w", err)
		}

		return nil
	}, nil
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

// IsPaused checks if all worker instances are paused
func (w *WorkerImpl) IsPaused(ctx context.Context) (bool, error) {
	// Create slice of worker IDs to check
	var workerIDs []string

	if w.nonDurableWorker != nil {
		mainID := w.nonDurableWorker.ID()
		workerIDs = append(workerIDs, *mainID)
	}

	if w.durableWorker != nil {
		durableID := w.durableWorker.ID()
		workerIDs = append(workerIDs, *durableID)
	}

	// If no workers exist, consider it not paused
	if len(workerIDs) == 0 {
		return false, nil
	}

	// Check pause status for all workers
	for _, id := range workerIDs {
		isPaused, err := w.workers.IsPaused(ctx, id)
		if err != nil {
			return false, err
		}

		// If any worker is not paused, return false
		if !isPaused {
			return false, nil
		}
	}

	// All workers are paused
	return true, nil
}

// Pause pauses all worker instances
func (w *WorkerImpl) Pause(ctx context.Context) error {
	// Pause main worker if it exists
	if w.nonDurableWorker != nil {
		_, err := w.workers.Pause(ctx, *w.nonDurableWorker.ID())
		if err != nil {
			return err
		}
	}

	// Pause durable worker if it exists
	if w.durableWorker != nil {
		_, err := w.workers.Pause(ctx, *w.durableWorker.ID())
		if err != nil {
			return err
		}
	}

	return nil
}

// Unpause resumes all paused worker instances
func (w *WorkerImpl) Unpause(ctx context.Context) error {
	// Unpause main worker if it exists
	if w.nonDurableWorker != nil {
		_, err := w.workers.Unpause(ctx, *w.nonDurableWorker.ID())
		if err != nil {
			return err
		}
	}

	// Unpause durable worker if it exists
	if w.durableWorker != nil {
		_, err := w.workers.Unpause(ctx, *w.durableWorker.ID())
		if err != nil {
			return err
		}
	}

	return nil
}
