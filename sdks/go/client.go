package hatchet

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
	"github.com/hatchet-dev/hatchet/sdks/go/internal"
)

// Client provides the main interface for interacting with Hatchet.
type Client struct {
	legacyClient v0Client.Client

	// Feature clients (lazy loaded)
	metrics    *features.MetricsClient
	rateLimits *features.RateLimitsClient
	crons      *features.CronsClient
	cel        *features.CELClient
	schedules  *features.SchedulesClient
	filters    *features.FiltersClient
	runs       *features.RunsClient
	workers    *features.WorkersClient
	workflows  *features.WorkflowsClient
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

// StandaloneTask represents a single task that runs independently without a workflow wrapper.
// It's essentially a specialized workflow containing only one task.
type StandaloneTask struct {
	name     string
	workflow *Workflow
	task     *Task
}

// GetName returns the name of the standalone task.
func (st *StandaloneTask) GetName() string {
	return st.name
}

// StandaloneTaskOption represents options that can be applied to standalone tasks.
// This interface allows both WorkflowOption and TaskOption to be used interchangeably.
type StandaloneTaskOption any

// NewStandaloneTask creates a standalone task that can be triggered independently.
// This is a specialized workflow containing only one task, making it easier to create
// simple single-task workflows without the workflow boilerplate.
//
// The function parameter must have the signature:
//
//	func(ctx hatchet.Context, input any) (any, error)
//
// Function signatures are validated at runtime using reflection.
//
// Options can be any combination of WorkflowOption and TaskOption.
func (c *Client) NewStandaloneTask(name string, fn any, options ...StandaloneTaskOption) *StandaloneTask {
	if name == "" {
		panic("standalone task name cannot be empty")
	}

	// Separate workflow and task options
	var workflowOptions []WorkflowOption
	var taskOptions []TaskOption

	for _, opt := range options {
		switch o := opt.(type) {
		case WorkflowOption:
			workflowOptions = append(workflowOptions, o)
		case TaskOption:
			taskOptions = append(taskOptions, o)
		default:
			panic("invalid option type for standalone task - must be WorkflowOption or TaskOption")
		}
	}

	// Create a workflow with the same name as the task
	workflow := c.NewWorkflow(name, workflowOptions...)

	// Create the single task within the workflow
	task := workflow.NewTask(name, fn, taskOptions...)

	return &StandaloneTask{
		name:     name,
		workflow: workflow,
		task:     task,
	}
}

// NewStandaloneDurableTask creates a standalone durable task that can be triggered independently.
// This is a specialized workflow containing only one durable task, making it easier to create
// simple single-task workflows with durable functionality.
//
// The function parameter must have the signature:
//
//	func(ctx hatchet.DurableContext, input any) (any, error)
//
// Function signatures are validated at runtime using reflection.
//
// Options can be any combination of WorkflowOption and TaskOption.
func (c *Client) NewStandaloneDurableTask(name string, fn any, options ...StandaloneTaskOption) *StandaloneTask {
	if name == "" {
		panic("standalone durable task name cannot be empty")
	}

	// Separate workflow and task options
	var workflowOptions []WorkflowOption
	var taskOptions []TaskOption

	for _, opt := range options {
		switch o := opt.(type) {
		case WorkflowOption:
			workflowOptions = append(workflowOptions, o)
		case TaskOption:
			taskOptions = append(taskOptions, o)
		default:
			panic("invalid option type for standalone durable task - must be WorkflowOption or TaskOption")
		}
	}

	// Create a workflow with the same name as the task
	workflow := c.NewWorkflow(name, workflowOptions...)

	// Create the single durable task within the workflow
	task := workflow.NewDurableTask(name, fn, taskOptions...)

	return &StandaloneTask{
		name:     name,
		workflow: workflow,
		task:     task,
	}
}

// Run executes the standalone task with the provided input and waits for completion.
func (st *StandaloneTask) Run(ctx context.Context, input any) (*TaskResult, error) {
	result, err := st.workflow.Run(ctx, input)
	if err != nil {
		return nil, err
	}

	// Extract the task result from the workflow result
	taskResult := result.TaskOutput(st.task.name)
	return taskResult, nil
}

// RunNoWait executes the standalone task with the provided input without waiting for completion.
// Returns a workflow run reference that can be used to track the run status.
func (st *StandaloneTask) RunNoWait(ctx context.Context, input any) (*WorkflowRef, error) {
	return st.workflow.RunNoWait(ctx, input)
}

// Dump implements the WorkflowBase interface for internal use, delegating to the underlying workflow.
func (st *StandaloneTask) Dump() (*contracts.CreateWorkflowVersionRequest, []internal.NamedFunction, []internal.NamedFunction, internal.WrappedTaskFn) {
	return st.workflow.Dump()
}

// WorkflowRef is a type that represents a reference to a workflow run.
type WorkflowRef struct {
	RunId string
}

// WorkflowResult wraps workflow execution results and provides type-safe conversion methods.
type WorkflowResult struct {
	result any
}

// TaskResult wraps a single task's output and provides type-safe conversion methods.
type TaskResult struct {
	result any
}

// TaskOutput extracts the output of a specific task from the workflow result.
// Returns a TaskResult that can be used to convert the task output into the desired type.
//
// Example usage:
//
//	taskResult := workflowResult.TaskOutput("myTask")
//	var output MyOutputType
//	err := taskResult.Into(&output)
func (wr *WorkflowResult) TaskOutput(taskName string) *TaskResult {
	// Handle different result structures that might come from workflow execution
	resultData := wr.result

	// Check if this is a raw v0Client.WorkflowResult that we need to extract from
	if workflowResult, ok := resultData.(*v0Client.WorkflowResult); ok {
		// Try to get the workflow results as a map
		results, err := workflowResult.Results()
		if err != nil {
			// Return empty TaskResult if we can't extract results
			return &TaskResult{result: nil}
		}
		resultData = results
	}

	// If the result is a map, look for the specific task
	if resultMap, ok := resultData.(map[string]any); ok {
		if taskOutput, exists := resultMap[taskName]; exists {
			return &TaskResult{result: taskOutput}
		}
	}

	// If we can't find the specific task, return the entire result
	// This handles cases where there's only one task
	return &TaskResult{result: resultData}
}

// Into converts the task result into the provided destination using JSON marshal/unmarshal.
// The destination should be a pointer to the desired type.
//
// Example usage:
//
//	var output MyOutputType
//	err := taskResult.Into(&output)
func (tr *TaskResult) Into(dest any) error {
	// Handle different result structures that might come from task execution
	resultData := tr.result

	// If the result is a pointer to interface{}, dereference it
	if ptr, ok := resultData.(*any); ok && ptr != nil {
		resultData = *ptr
	}

	// If the result is a pointer to string (JSON), unmarshal it directly
	if strPtr, ok := resultData.(*string); ok && strPtr != nil {
		return json.Unmarshal([]byte(*strPtr), dest)
	}

	// Convert the result to JSON and then unmarshal to destination
	jsonData, err := json.Marshal(resultData)
	if err != nil {
		return fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	if err := json.Unmarshal(jsonData, dest); err != nil {
		return fmt.Errorf("failed to unmarshal JSON to destination: %w", err)
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

// Metrics returns a feature client for interacting with workflow and task metrics.
func (c *Client) Metrics() *features.MetricsClient {
	if c.metrics == nil {
		tenantId := c.legacyClient.TenantId()
		c.metrics = features.NewMetricsClient(c.legacyClient.API(), tenantId)
	}

	return c.metrics
}

// RateLimits returns a client for managing rate limits.
func (c *Client) RateLimits() *features.RateLimitsClient {
	if c.rateLimits == nil {
		tenantId := c.legacyClient.TenantId()
		admin := c.legacyClient.Admin()
		c.rateLimits = features.NewRateLimitsClient(c.legacyClient.API(), tenantId, admin)
	}

	return c.rateLimits
}

// Runs returns a client for managing workflow runs.
func (c *Client) Runs() *features.RunsClient {
	if c.runs == nil {
		tenantId := c.legacyClient.TenantId()
		c.runs = features.NewRunsClient(c.legacyClient.API(), tenantId, c.legacyClient)
	}

	return c.runs
}

// Workers returns a client for managing workers.
func (c *Client) Workers() *features.WorkersClient {
	if c.workers == nil {
		tenantId := c.legacyClient.TenantId()
		c.workers = features.NewWorkersClient(c.legacyClient.API(), tenantId)
	}

	return c.workers
}

// Workflows returns a client for managing workflow definitions.
func (c *Client) Workflows() *features.WorkflowsClient {
	if c.workflows == nil {
		tenantId := c.legacyClient.TenantId()
		c.workflows = features.NewWorkflowsClient(c.legacyClient.API(), tenantId)
	}

	return c.workflows
}

// Crons returns a client for managing cron triggers.
func (c *Client) Crons() *features.CronsClient {
	if c.crons == nil {
		tenantId := c.legacyClient.TenantId()
		c.crons = features.NewCronsClient(c.legacyClient.API(), tenantId)
	}

	return c.crons
}

// CEL returns a client for working with CEL expressions.
func (c *Client) CEL() *features.CELClient {
	if c.cel == nil {
		tenantId := c.legacyClient.TenantId()
		c.cel = features.NewCELClient(c.legacyClient.API(), tenantId)
	}

	return c.cel
}

// Schedules returns a client for managing scheduled workflow runs.
func (c *Client) Schedules() *features.SchedulesClient {
	if c.schedules == nil {
		tenantId := c.legacyClient.TenantId()
		namespace := c.legacyClient.Namespace()
		c.schedules = features.NewSchedulesClient(c.legacyClient.API(), tenantId, &namespace)
	}

	return c.schedules
}

// Filters returns a client for managing event filters.
func (c *Client) Filters() *features.FiltersClient {
	if c.filters == nil {
		tenantId := c.legacyClient.TenantId()
		c.filters = features.NewFiltersClient(c.legacyClient.API(), tenantId)
	}

	return c.filters
}

// Events returns a client for sending and managing events.
func (c *Client) Events() v0Client.EventClient {
	return c.legacyClient.Event()
}
