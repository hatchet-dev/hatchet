package hatchet

import (
	"context"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/features"
	"github.com/hatchet-dev/hatchet/pkg/v1/worker"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
)

// Client provides the main interface for interacting with Hatchet.
type Client struct {
	v1Client v1.HatchetClient
}

// NewClient creates a new Hatchet client.
// Configuration options can be provided to customize the client behavior.
func NewClient(config ...v1.Config) (*Client, error) {
	v1Client, err := v1.NewHatchetClient(config...)
	if err != nil {
		return nil, err
	}

	return &Client{
		v1Client: v1Client,
	}, nil
}

// NewWorker creates a worker that can execute workflows.
// The worker is configured using functional options.
func (c *Client) NewWorker(name string, options ...WorkerOption) (worker.Worker, error) {
	config := &workerConfig{
		slots:        100,
		durableSlots: 1000,
	}

	for _, opt := range options {
		opt(config)
	}

	return c.v1Client.Worker(worker.WorkerOpts{
		Name:         name,
		Workflows:    config.workflows,
		Slots:        config.slots,
		Labels:       config.labels,
		Logger:       config.logger,
		LogLevel:     config.logLevel,
		DurableSlots: config.durableSlots,
	})
}

// NewWorkflow creates a new workflow definition.
// Workflows can be configured with triggers, events, and other options.
func (c *Client) NewWorkflow(name string, options ...WorkflowOption) *Workflow {
	return NewWorkflow(name, c.v1Client, options...)
}

// NewStandaloneTask creates a workflow containing a single task.
// This is a convenience method for simple workflows.
func (c *Client) NewStandaloneTask(name string, fn any, options ...TaskOption) *Workflow {
	return NewStandaloneTask(name, fn, c.v1Client, options...)
}

// Run executes a workflow with the provided input and waits for completion.
func (c *Client) Run(ctx context.Context, workflowName string, input any) error {
	_, err := c.v1Client.V0().Admin().RunWorkflow(workflowName, input)
	return err
}

// RunNoWait executes a workflow with the provided input without waiting for completion.
// Returns a workflow run reference that can be used to track the run status.
func (c *Client) RunNoWait(ctx context.Context, workflowName string, input any) (*v0Client.Workflow, error) {
	return c.v1Client.V0().Admin().RunWorkflow(workflowName, input)
}

// RunBulk executes multiple workflow instances with different inputs.
// Returns workflow run IDs that can be used to track the run statuses.
func (c *Client) RunBulk(ctx context.Context, workflowName string, inputs []any) ([]string, error) {
	workflows := make([]*v0Client.WorkflowRun, len(inputs))
	for i, input := range inputs {
		workflows[i] = &v0Client.WorkflowRun{
			Name:  workflowName,
			Input: input,
		}
	}
	return c.v1Client.V0().Admin().BulkRunWorkflow(workflows)
}

// Feature clients provide access to Hatchet's advanced functionality

// Metrics returns a client for interacting with workflow and task metrics.
func (c *Client) Metrics() features.MetricsClient {
	return c.v1Client.Metrics()
}

// RateLimits returns a client for managing rate limits.
func (c *Client) RateLimits() features.RateLimitsClient {
	return c.v1Client.RateLimits()
}

// Runs returns a client for managing workflow runs.
func (c *Client) Runs() features.RunsClient {
	return c.v1Client.Runs()
}

// Workers returns a client for managing workers.
func (c *Client) Workers() features.WorkersClient {
	return c.v1Client.Workers()
}

// Workflows returns a client for managing workflow definitions.
func (c *Client) Workflows() features.WorkflowsClient {
	return c.v1Client.Workflows()
}

// Crons returns a client for managing cron triggers.
func (c *Client) Crons() features.CronsClient {
	return c.v1Client.Crons()
}

// CEL returns a client for working with CEL expressions.
func (c *Client) CEL() features.CELClient {
	return c.v1Client.CEL()
}

// Schedules returns a client for managing scheduled workflow runs.
func (c *Client) Schedules() features.SchedulesClient {
	return c.v1Client.Schedules()
}

// Events returns a client for sending and managing events.
func (c *Client) Events() v0Client.EventClient {
	return c.v1Client.Events()
}

// Filters returns a client for managing event filters.
func (c *Client) Filters() features.FiltersClient {
	return c.v1Client.Filters()
}
