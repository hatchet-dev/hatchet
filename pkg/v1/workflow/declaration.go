// Package workflow provides functionality for defining, managing, and executing
// workflows in Hatchet. A workflow is a collection of tasks with defined
// dependencies and execution logic.
package workflow

import (
	"time"

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

// DurableWrappedTaskFn represents a durable task function that can be executed by the Hatchet worker.
// It takes a DurableHatchetContext and returns an interface{} result and an error.
type DurableWrappedTaskFn func(ctx worker.DurableHatchetContext) (interface{}, error)

// CreateOpts contains configuration options for creating a new workflow.
type CreateOpts[I any] struct {
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
	OnFailureTask *task.OnFailureTaskDeclaration[I]

	// (optional) Strategy for sticky execution of workflow runs
	StickyStrategy *types.StickyStrategy

	// (optional) Default settings for all tasks within this workflow
	TaskDefaults *task.TaskDefaults
}

// NamedFunction represents a function with its associated action ID
type NamedFunction struct {
	ActionID string
	Fn       WrappedTaskFn
}

// WorkflowBase defines the common interface for all workflow types.
type WorkflowBase interface {
	// Dump converts the workflow declaration into a protobuf request and function mappings.
	// Returns the workflow definition, regular task functions, durable task functions, and the on failure task function.
	Dump() (*contracts.CreateWorkflowVersionRequest, []NamedFunction, []NamedFunction, WrappedTaskFn)
}

// WorkflowDeclaration represents a workflow with input type I and output type O.
// It provides methods to define tasks, specify dependencies, and execute the workflow.
type WorkflowDeclaration[I any, O any] interface {
	WorkflowBase

	// Task registers a task that will be executed as part of the workflow
	Task(opts task.CreateOpts[I]) *task.TaskDeclaration[I]

	// DurableTask registers a durable task that will be executed as part of the workflow.
	// Durable tasks can be paused and resumed across workflow runs, making them suitable
	// for long-running operations or tasks that require human intervention.
	DurableTask(opts task.CreateOpts[I]) *task.DurableTaskDeclaration[I]

	// Run executes the workflow with the provided input.
	Run(input I) (*O, error)

	// TODO RunNoWait, Cron, Schedule
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
	OnFailureTask  *task.OnFailureTaskDeclaration[I]
	StickyStrategy *types.StickyStrategy

	TaskDefaults *task.TaskDefaults

	tasks        []*task.TaskDeclaration[I]
	durableTasks []*task.DurableTaskDeclaration[I]

	// Store task functions with their specific output types
	taskFuncs        map[string]interface{}
	durableTaskFuncs map[string]interface{}

	// Map to store task output setters
	outputSetters map[string]func(*O, interface{})
}

// NewWorkflowDeclaration creates a new workflow declaration with the specified options and client.
// The workflow will have input type I and output type O.
func NewWorkflowDeclaration[I any, O any](opts CreateOpts[I], v0 *v0Client.Client) WorkflowDeclaration[I, O] {
	wf := &workflowDeclarationImpl[I, O]{
		v0:               v0,
		Name:             opts.Name,
		OnEvents:         opts.OnEvents,
		OnCron:           opts.OnCron,
		Concurrency:      opts.Concurrency,
		OnFailureTask:    opts.OnFailureTask,
		StickyStrategy:   opts.StickyStrategy,
		TaskDefaults:     opts.TaskDefaults,
		tasks:            []*task.TaskDeclaration[I]{},
		taskFuncs:        make(map[string]interface{}),
		durableTasks:     []*task.DurableTaskDeclaration[I]{},
		durableTaskFuncs: make(map[string]interface{}),
		outputSetters:    make(map[string]func(*O, interface{})),
	}

	if opts.Version != "" {
		wf.Version = &opts.Version
	}

	if opts.Description != "" {
		wf.Description = &opts.Description
	}

	return wf
}

// Task registers a standard (non-durable) task with the workflow
func (w *workflowDeclarationImpl[I, O]) Task(opts task.CreateOpts[I]) *task.TaskDeclaration[I] {
	name := opts.Name

	fn := opts.Fn
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

	// Initialize pointers only for non-zero values
	var retryBackoffFactor *float32
	var retryMaxBackoffSeconds *int32
	var executionTimeout *time.Duration
	var scheduleTimeout *time.Duration
	var retries *int32

	if opts.RetryBackoffFactor != 0 {
		retryBackoffFactor = &opts.RetryBackoffFactor
	}
	if opts.RetryMaxBackoffSeconds != 0 {
		retryMaxBackoffSeconds = &opts.RetryMaxBackoffSeconds
	}
	if opts.ExecutionTimeout != 0 {
		executionTimeout = &opts.ExecutionTimeout
	}
	if opts.ScheduleTimeout != 0 {
		scheduleTimeout = &opts.ScheduleTimeout
	}
	if opts.Retries != 0 {
		retries = &opts.Retries
	}

	// Convert parent task declarations to parent task names
	parentNames := make([]string, len(opts.Parents))
	for i, parent := range opts.Parents {
		parentNames[i] = parent.Name
	}

	taskDecl := &task.TaskDeclaration[I]{
		Name:       opts.Name,
		Fn:         genericFn,
		Parents:    parentNames,
		Conditions: opts.Conditions,

		TaskShared: task.TaskShared{
			ExecutionTimeout:       executionTimeout,
			ScheduleTimeout:        scheduleTimeout,
			Retries:                retries,
			RetryBackoffFactor:     retryBackoffFactor,
			RetryMaxBackoffSeconds: retryMaxBackoffSeconds,
			RateLimits:             opts.RateLimits,
			WorkerLabels:           opts.WorkerLabels,
			Concurrency:            opts.Concurrency,
		},
	}

	w.tasks = append(w.tasks, taskDecl)
	w.taskFuncs[name] = fn

	return taskDecl
}

// DurableTask registers a durable task with the workflow
func (w *workflowDeclarationImpl[I, O]) DurableTask(opts task.CreateOpts[I]) *task.DurableTaskDeclaration[I] {
	name := opts.Name

	fn := opts.Fn
	// Use reflection to validate the function type
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func ||
		fnType.NumIn() != 2 ||
		fnType.NumOut() != 2 ||
		!fnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic("Invalid function type for durable task " + name + ": must be func(I, worker.DurableHatchetContext) (*T, error)")
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
	genericFn := func(input I, ctx worker.DurableHatchetContext) (*any, error) {
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

	// Initialize pointers only for non-zero values
	var retryBackoffFactor *float32
	var retryMaxBackoffSeconds *int32
	var executionTimeout *time.Duration
	var scheduleTimeout *time.Duration
	var retries *int32

	if opts.RetryBackoffFactor != 0 {
		retryBackoffFactor = &opts.RetryBackoffFactor
	}
	if opts.RetryMaxBackoffSeconds != 0 {
		retryMaxBackoffSeconds = &opts.RetryMaxBackoffSeconds
	}
	if opts.ExecutionTimeout != 0 {
		executionTimeout = &opts.ExecutionTimeout
	}
	if opts.ScheduleTimeout != 0 {
		scheduleTimeout = &opts.ScheduleTimeout
	}
	if opts.Retries != 0 {
		retries = &opts.Retries
	}

	// Convert parent task declarations to parent task names
	parentNames := make([]string, len(opts.Parents))
	for i, parent := range opts.Parents {
		parentNames[i] = parent.Name
	}

	taskDecl := &task.DurableTaskDeclaration[I]{
		Name:       opts.Name,
		Fn:         genericFn,
		Parents:    parentNames,
		Conditions: opts.Conditions,

		TaskShared: task.TaskShared{
			ExecutionTimeout:       executionTimeout,
			ScheduleTimeout:        scheduleTimeout,
			Retries:                retries,
			RetryBackoffFactor:     retryBackoffFactor,
			RetryMaxBackoffSeconds: retryMaxBackoffSeconds,
			RateLimits:             opts.RateLimits,
			WorkerLabels:           opts.WorkerLabels,
			Concurrency:            opts.Concurrency,
		},
	}

	w.durableTasks = append(w.durableTasks, taskDecl)
	w.durableTaskFuncs[name] = fn

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
func (w *workflowDeclarationImpl[I, O]) Dump() (*contracts.CreateWorkflowVersionRequest, []NamedFunction, []NamedFunction, WrappedTaskFn) {
	taskOpts := make([]*contracts.CreateTaskOpts, len(w.tasks))
	for i, task := range w.tasks {
		taskOpts[i] = task.Dump(w.Name, w.TaskDefaults)
	}

	durableOpts := make([]*contracts.CreateTaskOpts, len(w.durableTasks))
	for i, task := range w.durableTasks {
		durableOpts[i] = task.Dump(w.Name, w.TaskDefaults)
	}

	tasksToRegister := append(taskOpts, durableOpts...)

	req := &contracts.CreateWorkflowVersionRequest{
		Tasks:         tasksToRegister,
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

	// Create named function objects for regular tasks
	regularNamedFns := make([]NamedFunction, len(w.tasks))
	for i, task := range w.tasks {
		taskName := task.Name
		originalFn := w.taskFuncs[taskName]

		regularNamedFns[i] = NamedFunction{
			ActionID: taskOpts[i].Action,
			Fn: func(ctx worker.HatchetContext) (interface{}, error) {
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
			},
		}
	}

	// Create named function objects for durable tasks
	durableNamedFns := make([]NamedFunction, len(w.durableTasks))
	for i, task := range w.durableTasks {
		taskName := task.Name
		originalFn := w.durableTaskFuncs[taskName]

		durableNamedFns[i] = NamedFunction{
			ActionID: durableOpts[i].Action,
			Fn: func(ctx worker.HatchetContext) (interface{}, error) {
				var input I
				err := ctx.WorkflowInput(&input)
				if err != nil {
					return nil, err
				}

				// Create a DurableHatchetContext from the HatchetContext
				durableCtx := worker.NewDurableHatchetContext(ctx)

				// Call the original function using reflection
				fnValue := reflect.ValueOf(originalFn)
				inputs := []reflect.Value{reflect.ValueOf(input), reflect.ValueOf(durableCtx)}
				results := fnValue.Call(inputs)

				// Handle errors
				if !results[1].IsNil() {
					return nil, results[1].Interface().(error)
				}

				// Return the output
				return results[0].Interface(), nil
			},
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

			// Call the function using reflection
			fnValue := reflect.ValueOf(w.OnFailureTask.Fn)
			inputs := []reflect.Value{reflect.ValueOf(input), reflect.ValueOf(ctx)}
			results := fnValue.Call(inputs)

			// Handle errors
			if !results[1].IsNil() {
				return nil, results[1].Interface().(error)
			}

			// Get the result
			result := results[0].Interface()

			return result, nil
		}
	}

	return req, regularNamedFns, durableNamedFns, onFailureFn
}
