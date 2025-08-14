package hatchet

import (
	"context"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
	"github.com/hatchet-dev/hatchet/sdks/go/internal"
)

// Client provides the main interface for interacting with Hatchet.
type Client struct {
	legacyClient v0Client.Client
}

// NewClient creates a new Hatchet client.
// Configuration options can be provided to customize the client behavior.
//
// For examples of client usage, see:
//   - [Simple workflow](https://github.com/hatchet-dev/hatchet/tree/main/examples/go/v1/simple)
//   - [All examples](https://github.com/hatchet-dev/hatchet/tree/main/examples/go/v1)
func NewClient(opts ...v0Client.ClientOpt) (*Client, error) {
	legacyClient, err := v0Client.New(opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		legacyClient: legacyClient,
	}, nil
}

// Worker represents a worker that can execute workflows.
type Worker struct {
	nonDurable *worker.Worker
	durable    *worker.Worker
	name       string
}

// NewWorker creates a worker that can execute workflows.
func (c *Client) NewWorker(name string, options ...WorkerOption) (*Worker, error) {
	config := &workerConfig{
		slots:        100,
		durableSlots: 1000,
	}

	for _, opt := range options {
		opt(config)
	}

	workerOpts := []worker.WorkerOpt{
		worker.WithClient(c.legacyClient),
		worker.WithName(name),
		worker.WithMaxRuns(config.slots),
	}

	if config.logger != nil {
		workerOpts = append(workerOpts, worker.WithLogger(config.logger))
	}

	if config.logLevel != "" {
		workerOpts = append(workerOpts, worker.WithLogLevel(config.logLevel))
	}

	if config.labels != nil {
		workerOpts = append(workerOpts, worker.WithLabels(config.labels))
	}

	w, err := worker.NewWorker(workerOpts...)
	if err != nil {
		return nil, err
	}

	// Create the main non-durable worker
	nonDurableWorker := w
	var durableWorker *worker.Worker

	// Separate workflows into durable and non-durable
	var regularWorkflows []internal.WorkflowBase
	var durableWorkflows []internal.WorkflowBase

	for _, workflow := range config.workflows {
		_, _, durableActions, _ := workflow.Dump()
		hasDurableTasks := len(durableActions) > 0

		if hasDurableTasks {
			durableWorkflows = append(durableWorkflows, workflow)
		} else {
			regularWorkflows = append(regularWorkflows, workflow)
		}
	}

	// Register regular workflows with non-durable worker
	for _, workflow := range regularWorkflows {
		req, _, _, onFailureFn := workflow.Dump()
		err := nonDurableWorker.RegisterWorkflowV1(req)
		if err != nil {
			return nil, err
		}

		// Register on failure function if exists
		if req.OnFailureTask != nil && onFailureFn != nil {
			actionId := req.OnFailureTask.Action
			err = nonDurableWorker.RegisterAction(actionId, func(ctx worker.HatchetContext) (interface{}, error) {
				return onFailureFn(ctx)
			})
			if err != nil {
				return nil, err
			}
		}
	}

	// Create durable worker if needed
	if len(durableWorkflows) > 0 {
		durableWorkerOpts := []worker.WorkerOpt{
			worker.WithClient(c.legacyClient),
			worker.WithName(name + "-durable"),
			worker.WithMaxRuns(config.durableSlots),
		}

		if config.logger != nil {
			durableWorkerOpts = append(durableWorkerOpts, worker.WithLogger(config.logger))
		}

		if config.logLevel != "" {
			durableWorkerOpts = append(durableWorkerOpts, worker.WithLogLevel(config.logLevel))
		}

		if config.labels != nil {
			durableWorkerOpts = append(durableWorkerOpts, worker.WithLabels(config.labels))
		}

		durableWorker, err = worker.NewWorker(durableWorkerOpts...)
		if err != nil {
			return nil, err
		}

		// Register durable workflows with durable worker
		for _, workflow := range durableWorkflows {
			req, _, _, _ := workflow.Dump()
			err := durableWorker.RegisterWorkflowV1(req)
			if err != nil {
				return nil, err
			}

		}
	}

	return &Worker{
		nonDurable: nonDurableWorker,
		durable:    durableWorker,
		name:       name,
	}, nil
}

// Starts the worker instance and returns a cleanup function.
func (w *Worker) Start() (func() error, error) {
	// Start non-durable worker
	nonDurableCleanup, err := w.nonDurable.Start()
	if err != nil {
		return nil, err
	}

	var durableCleanup func() error
	// Start durable worker if it exists
	if w.durable != nil {
		durableCleanup, err = w.durable.Start()
		if err != nil {
			// Stop the non-durable worker if durable worker fails to start
			if nonDurableCleanup != nil {
				nonDurableCleanup()
			}
			return nil, err
		}
	}

	cleanup := func() error {
		if nonDurableCleanup != nil {
			nonDurableCleanup()
		}
		if durableCleanup != nil {
			durableCleanup()
		}
		return nil
	}

	return cleanup, nil
}

// StartBlocking starts both worker instances and blocks until they complete.
// This is a convenience method for common usage patterns.
func (w *Worker) StartBlocking() error {
	cleanup, err := w.Start()
	if err != nil {
		return err
	}

	// Wait for the cleanup function to complete
	return cleanup()
}

// NewWorkflow creates a new workflow definition.
// Workflows can be configured with triggers, events, and other options.
//
// For workflow examples, see:
//   - [DAG workflows](https://github.com/hatchet-dev/hatchet/tree/main/examples/go/v1/dag) - Complex workflows with dependencies
//   - [Event-driven workflows](https://github.com/hatchet-dev/hatchet/tree/main/examples/go/v1/events) - Event-triggered workflows
//   - [Cron scheduling](https://github.com/hatchet-dev/hatchet/tree/main/examples/go/v1/cron) - Scheduled workflows
//   - [All examples](https://github.com/hatchet-dev/hatchet/tree/main/examples/go/v1)
func (c *Client) NewWorkflow(name string, options ...WorkflowOption) *Workflow {
	return newWorkflow(name, c.legacyClient, options...)
}

// WorkflowRef is a type that represents a reference to a workflow run.
type WorkflowRef struct {
	RunId string
}

// Run executes a workflow with the provided input and waits for completion.
func (c *Client) Run(ctx context.Context, workflowName string, input any, opts ...RunOptFunc) (any, error) {
	v0Workflow, err := c.legacyClient.Admin().RunWorkflow(workflowName, input, opts...)
	if err != nil {
		return nil, err
	}

	result, err := v0Workflow.Result()
	if err != nil {
		return nil, err
	}

	return result.Results()
}

// RunNoWait executes a workflow with the provided input without waiting for completion.
// Returns a workflow run reference that can be used to track the run status.
func (c *Client) RunNoWait(ctx context.Context, workflowName string, input any, opts ...RunOptFunc) (*WorkflowRef, error) {
	res, err := c.legacyClient.Admin().RunWorkflow(workflowName, input, opts...)
	if err != nil {
		return nil, err
	}

	return &WorkflowRef{res.RunId()}, nil
}

// RunManyOpt is a type that represents the options for running multiple instances of a workflow with different inputs and options.
type RunManyOpt struct {
	Input any
	Opts  []RunOptFunc
}

// RunMany executes multiple workflow instances with different inputs.
// Returns workflow run IDs that can be used to track the run statuses.
func (c *Client) RunMany(ctx context.Context, workflowName string, inputs []RunManyOpt) ([]string, error) {
	workflows := make([]*v0Client.WorkflowRun, len(inputs))
	for i, input := range inputs {
		workflows[i] = &v0Client.WorkflowRun{
			Name:    workflowName,
			Input:   input.Input,
			Options: input.Opts,
		}
	}
	return c.legacyClient.Admin().BulkRunWorkflow(workflows)
}

// Feature clients provide access to Hatchet's advanced functionality

// Metrics returns a client for interacting with workflow and task metrics.
func (c *Client) Metrics() features.MetricsClient {
	tenantId := c.legacyClient.TenantId()
	return features.NewMetricsClient(c.legacyClient.API(), &tenantId)
}

// RateLimits returns a client for managing rate limits.
func (c *Client) RateLimits() features.RateLimitsClient {
	tenantId := c.legacyClient.TenantId()
	admin := c.legacyClient.Admin()
	return features.NewRateLimitsClient(c.legacyClient.API(), &tenantId, &admin)
}

// Runs returns a client for managing workflow runs.
func (c *Client) Runs() features.RunsClient {
	tenantId := c.legacyClient.TenantId()
	return features.NewRunsClient(c.legacyClient.API(), &tenantId, c.legacyClient)
}

// Workers returns a client for managing workers.
func (c *Client) Workers() features.WorkersClient {
	tenantId := c.legacyClient.TenantId()
	return features.NewWorkersClient(c.legacyClient.API(), &tenantId)
}

// Workflows returns a client for managing workflow definitions.
func (c *Client) Workflows() features.WorkflowsClient {
	tenantId := c.legacyClient.TenantId()
	return features.NewWorkflowsClient(c.legacyClient.API(), &tenantId)
}

// Crons returns a client for managing cron triggers.
func (c *Client) Crons() features.CronsClient {
	tenantId := c.legacyClient.TenantId()
	return features.NewCronsClient(c.legacyClient.API(), &tenantId)
}

// CEL returns a client for working with CEL expressions.
func (c *Client) CEL() features.CELClient {
	tenantId := c.legacyClient.TenantId()
	return features.NewCELClient(c.legacyClient.API(), &tenantId)
}

// Schedules returns a client for managing scheduled workflow runs.
func (c *Client) Schedules() features.SchedulesClient {
	tenantId := c.legacyClient.TenantId()
	namespace := c.legacyClient.Namespace()
	return features.NewSchedulesClient(c.legacyClient.API(), &tenantId, &namespace)
}

// Events returns a client for sending and managing events.
func (c *Client) Events() v0Client.EventClient {
	return c.legacyClient.Event()
}

// Filters returns a client for managing event filters.
func (c *Client) Filters() features.FiltersClient {
	tenantId := c.legacyClient.TenantId()
	return features.NewFiltersClient(c.legacyClient.API(), &tenantId)
}
