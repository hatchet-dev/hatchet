// package worker provides functionality for creating and managing hatchet workers.
// workers are responsible for executing workflow tasks and communicating with the hatchet API.
package worker

import (
	"time"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/rs/zerolog"
)

// WorkerLabels represents a map of labels that can be assigned to a worker
// for filtering and identification purposes.
type WorkerLabels map[string]interface{}

// CreateOpts defines the options for creating a new worker.
type CreateOpts struct {
	// the friendly name of the worker
	Name string

	// (optional) maximum number of concurrent runs on this worker
	Slots *int

	// (optional) labels to set on the worker
	Labels WorkerLabels

	// (optional) logger to use for the worker
	Logger *zerolog.Logger

	// (optional) log level
	LogLevel string
}

// Worker defines the interface for interacting with a hatchet worker.
type Worker interface {
	// Start begins worker execution in a non-blocking manner and returns a cleanup function.
	// the cleanup function should be called when the worker needs to be stopped.
	Start() (func() error, error)

	// StartBlocking begins worker execution and blocks until the process is interrupted.
	StartBlocking() error

	// RegisterWorkflows registers one or more workflows with the worker.
	RegisterWorkflows(workflows ...workflow.WorkflowBase) error
}

// WorkerImpl is the concrete implementation of the Worker interface.
type WorkerImpl struct {
	// v0 is the client used to communicate with the hatchet API.
	v0 *v0Client.Client

	// v0worker is the underlying worker implementation.
	v0worker *worker.Worker

	// name is the friendly name of the worker.
	name string

	// workflows is a slice of workflows registered with this worker.
	workflows []workflow.WorkflowBase

	// slots is the maximum number of concurrent runs allowed on this worker.
	slots int
}

// WithWorkflows is a functional option that configures a worker with the specified workflows.
func WithWorkflows(workflows ...workflow.WorkflowBase) func(*WorkerImpl) {
	return func(w *WorkerImpl) {
		w.workflows = workflows
	}
}

// NewWorker creates and configures a new Worker with the provided client and options.
// additional functional options can be provided to further customize the worker configuration.
// returns the created Worker interface and any error encountered during creation.
func NewWorker(client *v0Client.Client, opts CreateOpts, optFns ...func(*WorkerImpl)) (Worker, error) {
	w := &WorkerImpl{
		v0:   client,
		name: opts.Name,
	}

	for _, optFn := range optFns {
		optFn(w)
	}

	if opts.Slots == nil {
		w.slots = 100 // default to 100 slots
	}

	// create the worker
	worker, err := worker.NewWorker(
		worker.WithClient(*w.v0),
		worker.WithName(w.name),
		worker.WithMaxRuns(w.slots),
		worker.WithLogger(opts.Logger),
		worker.WithLogLevel(opts.LogLevel), // NOTE: must be called after WithLogger
		worker.WithLabels(map[string]interface{}(opts.Labels)),
	)

	if err != nil {
		return nil, err
	}

	w.v0worker = worker

	// register the workflows
	err = w.RegisterWorkflows(w.workflows...)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// RegisterWorkflows registers one or more workflows with the worker.
// it converts the workflows to the format expected by the underlying worker implementation
// and registers both the workflow definitions and their action functions.
// returns an error if registration fails.
func (w *WorkerImpl) RegisterWorkflows(workflows ...workflow.WorkflowBase) error {
	w.workflows = append(w.workflows, workflows...)

	for _, workflow := range workflows {
		dump, fns, onFailureFn := workflow.Dump()
		err := w.v0worker.RegisterWorkflowV1(dump)
		if err != nil {
			return err
		}
		for i, fn := range fns {
			actionId := dump.Tasks[i].Action
			err := w.v0worker.RegisterAction(actionId, fn)
			if err != nil {
				return err
			}
		}

		if dump.OnFailureTask != nil {
			actionId := dump.OnFailureTask.Action
			err := w.v0worker.RegisterAction(actionId, onFailureFn)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Start begins worker execution in a non-blocking manner.
// returns a cleanup function to be called when the worker should be stopped,
// and any error encountered during startup.
func (w *WorkerImpl) Start() (func() error, error) {
	return w.v0worker.Start()
}

// StartBlocking begins worker execution and blocks until the process is interrupted.
// this method handles graceful shutdown via interrupt signals.
// returns any error encountered during startup or shutdown.
func (w *WorkerImpl) StartBlocking() error {
	cleanup, err := w.Start()
	if err != nil {
		return err
	}
	ch := cmdutils.InterruptChan()
	interruptCtx, cancel := cmdutils.InterruptContextFromChan(ch)
	defer cancel()

	for {
		select {
		case <-interruptCtx.Done():
			err := cleanup()
			if err != nil {
				panic(err)
			}
			return nil
		default:
			time.Sleep(time.Second)
		}
	}
}
