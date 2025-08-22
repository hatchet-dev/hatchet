package hatchet

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-viper/mapstructure/v2"
	"golang.org/x/sync/errgroup"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

// Client provides the main interface for interacting with Hatchet.
type Client struct {
	legacyClient v0Client.Client
}

// NewClient creates a new Hatchet client.
// Configuration options can be provided to customize the client behavior.
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

	if config.labels != nil {
		workerOpts = append(workerOpts, worker.WithLabels(config.labels))
	}

	nonDurableWorker, err := worker.NewWorker(workerOpts...)
	if err != nil {
		return nil, err
	}

	var durableWorker *worker.Worker

	for _, workflow := range config.workflows {
		req, regularActions, durableActions, onFailureFn := workflow.Dump()
		hasDurableTasks := len(durableActions) > 0

		if hasDurableTasks {
			if durableWorker == nil {
				durableWorkerOpts := workerOpts
				durableWorkerOpts = append(durableWorkerOpts, worker.WithName(name+"-durable"))
				durableWorkerOpts = append(durableWorkerOpts, worker.WithMaxRuns(config.durableSlots))

				durableWorker, err = worker.NewWorker(durableWorkerOpts...)
				if err != nil {
					return nil, err
				}
			}

			err := durableWorker.RegisterWorkflowV1(req)
			if err != nil {
				return nil, err
			}
		} else {
			err := nonDurableWorker.RegisterWorkflowV1(req)
			if err != nil {
				return nil, err
			}
		}

		for _, namedFn := range durableActions {
			err = durableWorker.RegisterAction(namedFn.ActionID, namedFn.Fn)
			if err != nil {
				return nil, err
			}
		}

		for _, namedFn := range regularActions {
			err = nonDurableWorker.RegisterAction(namedFn.ActionID, namedFn.Fn)
			if err != nil {
				return nil, err
			}
		}

		// Register on failure function if exists
		if req.OnFailureTask != nil && onFailureFn != nil {
			actionId := req.OnFailureTask.Action
			err = nonDurableWorker.RegisterAction(actionId, func(ctx worker.HatchetContext) (any, error) {
				return onFailureFn(ctx)
			})
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
	var workers []*worker.Worker

	if w.nonDurable != nil {
		workers = append(workers, w.nonDurable)
	}

	if w.durable != nil {
		workers = append(workers, w.durable)
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

// StartBlocking starts the worker and blocks until it completes.
// This is a convenience method for common usage patterns.
func (w *Worker) StartBlocking(ctx context.Context) error {
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

// NewWorkflow creates a new workflow definition.
// Workflows can be configured with triggers, events, and other options.
func (c *Client) NewWorkflow(name string, options ...WorkflowOption) *Workflow {
	return newWorkflow(name, c.legacyClient, options...)
}

// WorkflowRef is a type that represents a reference to a workflow run.
type WorkflowRef struct {
	RunId string
}

// WorkflowResult wraps workflow execution results and provides type-safe conversion methods.
type WorkflowResult struct {
	result any
}

// Into converts the workflow result into the provided destination using mapstructure.
// The destination should be a pointer to the desired type.
//
// Example usage:
//
//	var output MyOutputType
//	err := result.Into(&output)
func (wr *WorkflowResult) Into(dest any) error {
	// Handle different result structures that might come from workflow execution
	resultData := wr.result

	// Check if this is a raw v0Client.WorkflowResult that we need to extract from
	if workflowResult, ok := resultData.(*v0Client.WorkflowResult); ok {
		// For workflows with a single task, extract the first (and likely only) task result
		// Try to get the workflow results as a map
		results, err := workflowResult.Results()
		if err != nil {
			return fmt.Errorf("failed to get workflow results: %w", err)
		}
		resultData = results
	}

	// If the result is a pointer to interface{}, dereference it
	if ptr, ok := resultData.(*any); ok && ptr != nil {
		resultData = *ptr
	}

	// If the result is a map, it might contain step results - try to extract the main result
	if resultMap, ok := resultData.(map[string]any); ok {
		// For workflows with a single task output, try to find the task result
		// First, check if there's a direct value that matches our destination type
		if len(resultMap) == 1 {
			for _, value := range resultMap {
				resultData = value
				// If the value is a pointer to interface{}, dereference it
				if ptr, ok := resultData.(*any); ok && ptr != nil {
					resultData = *ptr
				}
				// If the value is a pointer to string (JSON), unmarshal it
				if strPtr, ok := resultData.(*string); ok && strPtr != nil {
					var unmarshaled any
					if err := json.Unmarshal([]byte(*strPtr), &unmarshaled); err != nil {
						return fmt.Errorf("failed to unmarshal JSON result: %w", err)
					}
					resultData = unmarshaled
				}
				break
			}
		}
	}

	config := &mapstructure.DecoderConfig{
		TagName:          "json",
		Result:           dest,
		WeaklyTypedInput: true,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(resultData); err != nil {
		return fmt.Errorf("failed to decode result: %w", err)
	}

	return nil
}

// Raw returns the raw workflow result as interface{}.
func (wr *WorkflowResult) Raw() any {
	return wr.result
}

// Run executes a workflow with the provided input and waits for completion.
func (c *Client) Run(ctx context.Context, workflowName string, input any, opts ...RunOptFunc) (*WorkflowResult, error) {
	v0Workflow, err := c.legacyClient.Admin().RunWorkflow(workflowName, input, opts...)
	if err != nil {
		return nil, err
	}

	result, err := v0Workflow.Result()
	if err != nil {
		return nil, err
	}

	workflowResult, err := result.Results()
	if err != nil {
		return nil, err
	}

	return &WorkflowResult{result: workflowResult}, nil
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
