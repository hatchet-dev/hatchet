package hatchet

import (
	"context"
	"fmt"
	"strings"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

// Client provides the main interface for interacting with Hatchet.
type Client struct {
	v0Client v0Client.Client
}

// NewClient creates a new Hatchet client.
// Configuration options can be provided to customize the client behavior.
func NewClient(opts ...v0Client.ClientOpt) (*Client, error) {
	v0Client, err := v0Client.New(opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		v0Client: v0Client,
	}, nil
}

// NewWorker creates a worker that can execute workflows.
// The worker is configured using functional options.
func (c *Client) NewWorker(name string, options ...WorkerOption) (*worker.Worker, error) {
	config := &workerConfig{
		slots:        100,
		durableSlots: 1000,
	}

	for _, opt := range options {
		opt(config)
	}

	// Use the higher of slots or durableSlots as the max runs
	// since durable tasks tend to be longer-running
	maxRuns := config.slots
	if config.durableSlots > maxRuns {
		maxRuns = config.durableSlots
	}

	workerOpts := []worker.WorkerOpt{
		worker.WithClient(c.v0Client),
		worker.WithName(name),
		worker.WithMaxRuns(maxRuns),
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

	// Register workflows with the worker
	for _, workflow := range config.workflows {
		req, regularFns, durableFns, onFailureFn := workflow.Dump()
		err := w.RegisterWorkflowV1(req)
		if err != nil {
			return nil, err
		}

		// Register regular task functions
		for _, fn := range regularFns {
			err = w.RegisterAction(fn.ActionID, func(ctx worker.HatchetContext) (interface{}, error) {
				return fn.Fn(ctx)
			})
			if err != nil {
				return nil, err
			}
		}

		// Register durable task functions
		for _, fn := range durableFns {
			err = w.RegisterAction(fn.ActionID, func(ctx worker.HatchetContext) (interface{}, error) {
				return fn.Fn(ctx)
			})
			if err != nil {
				return nil, err
			}
		}

		// Register on failure function if exists
		if onFailureFn != nil {
			// The on-failure action ID is derived from the workflow name
			onFailureActionID := strings.ToLower(fmt.Sprintf("%s:on-failure", req.Name))
			err = w.RegisterAction(onFailureActionID, func(ctx worker.HatchetContext) (interface{}, error) {
				return onFailureFn(ctx)
			})
			if err != nil {
				return nil, err
			}
		}
	}

	return w, nil
}

// NewWorkflow creates a new workflow definition.
// Workflows can be configured with triggers, events, and other options.
func (c *Client) NewWorkflow(name string, options ...WorkflowOption) *Workflow {
	return NewWorkflow(name, c.v0Client, options...)
}

// NewStandaloneTask creates a workflow containing a single task.
// This is a convenience method for simple workflows.
func (c *Client) NewStandaloneTask(name string, fn any, options ...TaskOption) *Workflow {
	return NewStandaloneTask(name, fn, c.v0Client, options...)
}

// Run executes a workflow with the provided input and waits for completion.
func (c *Client) Run(ctx context.Context, workflowName string, input any) error {
	_, err := c.v0Client.Admin().RunWorkflow(workflowName, input)
	return err
}

// RunWithPriority executes a workflow with the provided input and priority, waits for completion.
func (c *Client) RunWithPriority(ctx context.Context, workflowName string, input any, priority int32) error {
	_, err := c.v0Client.Admin().RunWorkflow(workflowName, input, v0Client.WithPriority(priority))
	return err
}

// RunNoWait executes a workflow with the provided input without waiting for completion.
// Returns a workflow run reference that can be used to track the run status.
func (c *Client) RunNoWait(ctx context.Context, workflowName string, input any) (*v0Client.Workflow, error) {
	return c.v0Client.Admin().RunWorkflow(workflowName, input)
}

// RunMany executes multiple workflow instances with different inputs.
// Returns workflow run IDs that can be used to track the run statuses.
func (c *Client) RunMany(ctx context.Context, workflowName string, inputs []any) ([]string, error) {
	workflows := make([]*v0Client.WorkflowRun, len(inputs))
	for i, input := range inputs {
		workflows[i] = &v0Client.WorkflowRun{
			Name:  workflowName,
			Input: input,
		}
	}
	return c.v0Client.Admin().BulkRunWorkflow(workflows)
}

// Feature clients provide access to Hatchet's advanced functionality

// Metrics returns a client for interacting with workflow and task metrics.
func (c *Client) Metrics() features.MetricsClient {
	tenantId := c.v0Client.TenantId()
	return features.NewMetricsClient(c.v0Client.API(), &tenantId)
}

// RateLimits returns a client for managing rate limits.
func (c *Client) RateLimits() features.RateLimitsClient {
	tenantId := c.v0Client.TenantId()
	admin := c.v0Client.Admin()
	return features.NewRateLimitsClient(c.v0Client.API(), &tenantId, &admin)
}

// Runs returns a client for managing workflow runs.
func (c *Client) Runs() features.RunsClient {
	tenantId := c.v0Client.TenantId()
	return features.NewRunsClient(c.v0Client.API(), &tenantId, c.v0Client)
}

// Workers returns a client for managing workers.
func (c *Client) Workers() features.WorkersClient {
	tenantId := c.v0Client.TenantId()
	return features.NewWorkersClient(c.v0Client.API(), &tenantId)
}

// Workflows returns a client for managing workflow definitions.
func (c *Client) Workflows() features.WorkflowsClient {
	tenantId := c.v0Client.TenantId()
	return features.NewWorkflowsClient(c.v0Client.API(), &tenantId)
}

// Crons returns a client for managing cron triggers.
func (c *Client) Crons() features.CronsClient {
	tenantId := c.v0Client.TenantId()
	return features.NewCronsClient(c.v0Client.API(), &tenantId)
}

// CEL returns a client for working with CEL expressions.
func (c *Client) CEL() features.CELClient {
	tenantId := c.v0Client.TenantId()
	return features.NewCELClient(c.v0Client.API(), &tenantId)
}

// Schedules returns a client for managing scheduled workflow runs.
func (c *Client) Schedules() features.SchedulesClient {
	tenantId := c.v0Client.TenantId()
	namespace := c.v0Client.Namespace()
	return features.NewSchedulesClient(c.v0Client.API(), &tenantId, &namespace)
}

// Events returns a client for sending and managing events.
func (c *Client) Events() v0Client.EventClient {
	return c.v0Client.Event()
}

// Filters returns a client for managing event filters.
func (c *Client) Filters() features.FiltersClient {
	tenantId := c.v0Client.TenantId()
	return features.NewFiltersClient(c.v0Client.API(), &tenantId)
}
