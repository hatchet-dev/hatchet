package hatchet

import (
	"context"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/worker"
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

// Run executes a workflow with the provided input.
func (c *Client) Run(ctx context.Context, workflowName string, input any) error {
	_, err := c.v1Client.V0().Admin().RunWorkflow(workflowName, input)
	return err
}
