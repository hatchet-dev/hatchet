// Package workflow provides functionality for defining, managing, and executing
// workflows in Hatchet. A workflow is a collection of tasks with defined
// dependencies and execution logic.
package workflow

import (
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/worker"

	"reflect"

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

	// Task registers a task that will be executed as part of the workflow
	Task(opts task.CreateOpts[I], fn interface{}) *task.TaskDeclaration[I, any]

	// Run executes the workflow with the provided input.
	Run(input I) (*O, error)
}

// Define a TaskDeclaration with specific output type
type TaskWithSpecificOutput[I any, T any] struct {
	Name string
	Fn   func(input I, ctx worker.HatchetContext) (*T, error)
	// other fields
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

	tasks []*task.TaskDeclaration[I, any]

	// Store task functions with their specific output types
	taskFuncs map[string]interface{}

	// Map to store task output setters
	outputSetters map[string]func(*O, interface{})
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
		tasks:          []*task.TaskDeclaration[I, any]{},
		taskFuncs:      make(map[string]interface{}),
		outputSetters:  make(map[string]func(*O, interface{})),
	}

	if opts.Version != "" {
		wf.Version = &opts.Version
	}

	if opts.Description != "" {
		wf.Description = &opts.Description
	}

	return wf
}

// TaskOutput registers a task with a specific output type and how to map it to the final result
func (w *workflowDeclarationImpl[I, O]) Task(opts task.CreateOpts[I], fn interface{}) *task.TaskDeclaration[I, any] {

	name := opts.Name
	// Use reflection to validate the function type
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func ||
		fnType.NumIn() != 2 ||
		fnType.NumOut() != 2 ||
		!fnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic("Invalid function type for task " + name + ": must be func(I, worker.HatchetContext) (*T, error)")
	}

	// Create a setter function that can set this specific output type to the corresponding field in O
	w.outputSetters[name] = func(result *O, output interface{}) {
		resultValue := reflect.ValueOf(result).Elem()
		field := resultValue.FieldByName(name)

		if field.IsValid() && field.CanSet() {
			outputValue := reflect.ValueOf(output).Elem()
			field.Set(outputValue)
		}
	}

	// Create a generic task function that wraps the specific one
	genericFn := func(input I, ctx worker.HatchetContext) (*any, error) {
		// Use reflection to call the specific function
		fnValue := reflect.ValueOf(fn)
		inputs := []reflect.Value{reflect.ValueOf(input), reflect.ValueOf(ctx)}
		results := fnValue.Call(inputs)

		// Handle errors
		if !results[1].IsNil() {
			return nil, results[1].Interface().(error)
		}

		// Return the output as any
		output := results[0].Interface()
		return &output, nil
	}

	taskDecl := task.NewTaskDeclaration(opts, genericFn)
	w.tasks = append(w.tasks, taskDecl)
	w.taskFuncs[name] = fn

	return taskDecl
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

	// Create wrapper functions for each task
	fns := make([]WrappedTaskFn, len(w.tasks))
	for i, task := range w.tasks {
		taskName := task.Name
		originalFn := w.taskFuncs[taskName]

		fns[i] = func(ctx worker.HatchetContext) (interface{}, error) {
			var input I
			err := ctx.WorkflowInput(&input)
			if err != nil {
				return nil, err
			}

			// Call the original function using reflection
			fnValue := reflect.ValueOf(originalFn)
			inputs := []reflect.Value{reflect.ValueOf(input), reflect.ValueOf(ctx)}
			results := fnValue.Call(inputs)

			// Handle errors
			if !results[1].IsNil() {
				return nil, results[1].Interface().(error)
			}

			// Return the output
			return results[0].Interface(), nil
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

// When executing a task, use type assertions to handle the specific output
