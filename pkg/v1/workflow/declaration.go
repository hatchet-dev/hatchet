// Package workflow provides functionality for defining, managing, and executing
// workflows in Hatchet. A workflow is a collection of tasks with defined
// dependencies and execution logic.
package workflow

import (
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/worker"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// WrappedTaskFn represents a task function that can be executed by the Hatchet worker.
// It takes a HatchetContext and returns an interface{} result and an error.
type WrappedTaskFn func(ctx worker.HatchetContext) (interface{}, error)

// CreateOpts contains configuration options for creating a new workflow.
type CreateOpts struct {
	// (required) The friendly name of the workflow
	Name string

	// (optional) The version of the workflow
	Version string

	// (optional) The human-readable description of the workflow
	Description string

	// (optional) The event names that trigger the workflow
	OnEvents []string

	// (optional) The cron expressions for scheduled workflow runs
	OnCron []string

	// (optional) Concurrency settings to control parallel execution
	Concurrency *types.Concurrency

	// (optional) Task to execute when workflow fails
	OnFailureTask *task.OnFailureTaskDeclaration[any, any]

	// (optional) Strategy for sticky execution of workflow runs
	StickyStrategy *types.StickyStrategy

	// (optional) Default settings for all tasks within this workflow
	TaskDefaults *task.TaskDefaults
}

// WorkflowBase defines the common interface for all workflow types.
type WorkflowBase interface {
	// Dump converts the workflow declaration into a protobuf request and function mappings.
	// Returns the workflow definition, the task functions, and the on failure task function.
	Dump() (*contracts.CreateWorkflowVersionRequest, []WrappedTaskFn, WrappedTaskFn)
}

// WorkflowDeclaration represents a workflow with input type I and output type O.
// It provides methods to define tasks, specify dependencies, and execute the workflow.
type WorkflowDeclaration[I any, O any] interface {
	WorkflowBase

	// Task creates and adds a new task to the workflow based on the provided options.
	// Returns a pointer to the created task declaration for future reference.
	Task(opts task.CreateOpts[I, O]) *task.TaskDeclaration[I, O]

	// Run executes the workflow with the provided input.
	// Returns the workflow output and any error encountered during execution.
	Run(input I) (*O, error)
}

// workflowDeclarationImpl is the concrete implementation of WorkflowDeclaration.
// It contains all the data and logic needed to define and execute a workflow.
type workflowDeclarationImpl[I any, O any] struct {
	v0 *v0Client.Client

	Name           string
	Version        *string
	Description    *string
	OnEvents       []string
	OnCron         []string
	Concurrency    *types.Concurrency
	OnFailureTask  *task.OnFailureTaskDeclaration[any, any]
	StickyStrategy *types.StickyStrategy

	TaskDefaults *task.TaskDefaults

	tasks []*task.TaskDeclaration[I, O]
}

// NewWorkflowDeclaration creates a new workflow declaration with the specified options and client.
// The workflow will have input type I and output type O.
func NewWorkflowDeclaration[I any, O any](opts CreateOpts, v0 *v0Client.Client) WorkflowDeclaration[I, O] {
	wf := &workflowDeclarationImpl[I, O]{
		v0:             v0,
		Name:           opts.Name,
		OnEvents:       opts.OnEvents,
		OnCron:         opts.OnCron,
		Concurrency:    opts.Concurrency,
		OnFailureTask:  opts.OnFailureTask,
		StickyStrategy: opts.StickyStrategy,
		TaskDefaults:   opts.TaskDefaults,
		tasks:          []*task.TaskDeclaration[I, O]{},
	}

	if opts.Version != "" {
		wf.Version = &opts.Version
	}

	if opts.Description != "" {
		wf.Description = &opts.Description
	}

	return wf
}

// Task creates a new task declaration with the provided options and adds it to the workflow.
// Returns a pointer to the created task declaration for future reference.
func (w *workflowDeclarationImpl[I, O]) Task(opts task.CreateOpts[I, O]) *task.TaskDeclaration[I, O] {
	task := task.NewTaskDeclaration(opts)
	w.tasks = append(w.tasks, task)
	return task
}

// Run executes the workflow with the provided input.
// It triggers a workflow run via the Hatchet client and waits for the result.
// Returns the workflow output and any error encountered during execution.
func (w *workflowDeclarationImpl[I, O]) Run(input I) (*O, error) {
	// TODO run opts
	run, err := (*w.v0).Admin().RunWorkflow(w.Name, input)

	if err != nil {
		return nil, err
	}

	// TODO the result method does not work as expect at this time
	_, err = run.Result()

	if err != nil {
		return nil, err
	}

	return nil, nil
}

// Dump converts the workflow declaration into a protobuf request and function mappings.
// This is used to serialize the workflow for transmission to the Hatchet server.
// Returns the workflow definition as a protobuf request, the task functions, and the on-failure task function.
func (w *workflowDeclarationImpl[I, O]) Dump() (*contracts.CreateWorkflowVersionRequest, []WrappedTaskFn, WrappedTaskFn) {
	taskOpts := make([]*contracts.CreateTaskOpts, len(w.tasks))
	for i, task := range w.tasks {
		taskOpts[i] = task.Dump(w.Name, w.TaskDefaults)
	}

	req := &contracts.CreateWorkflowVersionRequest{
		Tasks: taskOpts,

		Name:          w.Name,
		EventTriggers: w.OnEvents,
		CronTriggers:  w.OnCron,
	}

	if w.Version != nil {
		req.Version = *w.Version
	}

	if w.Description != nil {
		req.Description = *w.Description
	}

	if w.Concurrency != nil {
		req.Concurrency = &contracts.Concurrency{
			Expression: w.Concurrency.Expression,
			MaxRuns:    w.Concurrency.MaxRuns,
		}

		if w.Concurrency.LimitStrategy != nil {
			strategy := *w.Concurrency.LimitStrategy
			strategyInt := contracts.ConcurrencyLimitStrategy_value[string(strategy)]
			strategyEnum := contracts.ConcurrencyLimitStrategy(strategyInt)
			req.Concurrency.LimitStrategy = &strategyEnum
		}
	}

	if w.OnFailureTask != nil {
		req.OnFailureTask = w.OnFailureTask.Dump(w.Name, w.TaskDefaults)
	}

	if w.StickyStrategy != nil {
		stickyStrategy := contracts.StickyStrategy(*w.StickyStrategy)
		req.Sticky = &stickyStrategy
	}

	// wrap the v1 task functions to be compatible with the v0 worker
	fns := make([]WrappedTaskFn, len(w.tasks))
	for i, task := range w.tasks {
		t := task // Create a local copy to avoid closure issues
		fns[i] = func(ctx worker.HatchetContext) (interface{}, error) {
			var input I
			err := ctx.WorkflowInput(&input)
			if err != nil {
				return nil, err
			}

			result, err := t.Fn(input, ctx)
			if err != nil {
				return nil, err
			}

			return result, nil
		}
	}

	var onFailureFn WrappedTaskFn
	if w.OnFailureTask != nil {
		onFailureFn = func(ctx worker.HatchetContext) (interface{}, error) {
			var input I
			err := ctx.WorkflowInput(&input)
			if err != nil {
				return nil, err
			}

			result, err := w.OnFailureTask.Fn(input, ctx)
			if err != nil {
				return nil, err
			}

			return result, nil
		}
	}

	return req, fns, onFailureFn
}
