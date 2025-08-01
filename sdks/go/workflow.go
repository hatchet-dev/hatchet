package hatchet

import (
	"context"
	"reflect"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// Workflow represents a workflow definition that can contain multiple tasks.
type Workflow struct {
	declaration workflow.WorkflowDeclaration[any, any]
	v1Client    v1.HatchetClient
}

// WorkflowOption configures a workflow instance.
type WorkflowOption func(*workflowConfig)

type workflowConfig struct {
	onCron         []string
	onEvents       []string
	defaultFilters []types.DefaultFilter
	concurrency    []types.Concurrency
	version        string
	description    string
}

// WithWorkflowCron configures the workflow to run on a cron schedule.
// Multiple cron expressions can be provided.
func WithWorkflowCron(cronExpressions ...string) WorkflowOption {
	return func(config *workflowConfig) {
		config.onCron = cronExpressions
	}
}

// WithWorkflowEvents configures the workflow to trigger on specific events.
func WithWorkflowEvents(events ...string) WorkflowOption {
	return func(config *workflowConfig) {
		config.onEvents = events
	}
}

// WithWorkflowVersion sets the version identifier for the workflow.
func WithWorkflowVersion(version string) WorkflowOption {
	return func(config *workflowConfig) {
		config.version = version
	}
}

// WithWorkflowDescription sets a human-readable description for the workflow.
func WithWorkflowDescription(description string) WorkflowOption {
	return func(config *workflowConfig) {
		config.description = description
	}
}

// NewWorkflow creates a new workflow definition.
func NewWorkflow(name string, v1Client v1.HatchetClient, options ...WorkflowOption) *Workflow {
	config := &workflowConfig{}

	for _, opt := range options {
		opt(config)
	}

	declaration := workflow.NewWorkflowDeclaration[any, any](
		create.WorkflowCreateOpts[any]{
			Name:        name,
			Version:     config.version,
			Description: config.description,
			OnEvents:    config.onEvents,
			OnCron:      config.onCron,
		},
		v1Client.V0(),
	)

	return &Workflow{
		declaration: declaration,
		v1Client:    v1Client,
	}
}

// TaskOption configures a task instance.
type TaskOption func(*taskConfig)

type taskConfig struct {
	retries               int32
	retryBackoffFactor    float32
	retryMaxBackoffSeconds int32
	executionTimeout      time.Duration
	onCron               []string
	onEvents             []string
	defaultFilters       []types.DefaultFilter
	concurrency          []*types.Concurrency
	rateLimits           []*types.RateLimit
	isDurable            bool
	parents              []create.NamedTask
	waitFor              condition.Condition
	skipIf               condition.Condition
}

// WithRetries sets the number of retry attempts for failed tasks.
func WithRetries(retries int) TaskOption {
	return func(config *taskConfig) {
		config.retries = int32(retries)
	}
}

// WithRetryBackoff configures exponential backoff for task retries.
func WithRetryBackoff(factor float32, maxBackoffSeconds int) TaskOption {
	return func(config *taskConfig) {
		config.retryBackoffFactor = factor
		config.retryMaxBackoffSeconds = int32(maxBackoffSeconds)
	}
}

// WithTimeout sets the maximum execution duration for a task.
func WithTimeout(timeout time.Duration) TaskOption {
	return func(config *taskConfig) {
		config.executionTimeout = timeout
	}
}

// WithCron configures standalone tasks to run on a cron schedule.
// Only applicable to standalone tasks, not workflow tasks.
func WithCron(cronExpressions ...string) TaskOption {
	return func(config *taskConfig) {
		config.onCron = cronExpressions
	}
}

// WithEvents configures standalone tasks to trigger on specific events.
// Only applicable to standalone tasks, not workflow tasks.
func WithEvents(events ...string) TaskOption {
	return func(config *taskConfig) {
		config.onEvents = events
	}
}

// WithFilters sets default filters for event-triggered tasks.
func WithFilters(filters ...types.DefaultFilter) TaskOption {
	return func(config *taskConfig) {
		config.defaultFilters = filters
	}
}

// WithConcurrency sets concurrency limits for task execution.
func WithConcurrency(concurrency ...*types.Concurrency) TaskOption {
	return func(config *taskConfig) {
		config.concurrency = concurrency
	}
}

// WithDurable marks a task as durable, enabling persistent state and long-running operations.
func WithDurable() TaskOption {
	return func(config *taskConfig) {
		config.isDurable = true
	}
}

// WithRateLimits sets rate limiting for task execution.
func WithRateLimits(rateLimits ...*types.RateLimit) TaskOption {
	return func(config *taskConfig) {
		config.rateLimits = rateLimits
	}
}

// WithParents sets parent task dependencies.
func WithParents(parents ...create.NamedTask) TaskOption {
	return func(config *taskConfig) {
		config.parents = parents
	}
}

// WithWaitFor sets a condition that must be met before the task executes.
func WithWaitFor(condition condition.Condition) TaskOption {
	return func(config *taskConfig) {
		config.waitFor = condition
	}
}

// WithSkipIf sets a condition that will skip the task if met.
func WithSkipIf(condition condition.Condition) TaskOption {
	return func(config *taskConfig) {
		config.skipIf = condition
	}
}

// Task represents a task reference for building DAGs.
type Task struct {
	NamedTask create.NamedTask
}

// NewTask adds a task to the workflow.
//
// The function parameter must have the signature:
//   func(ctx Context, input T) (T, error)
//
// For durable tasks, use:
//   func(ctx DurableContext, input T) (T, error)
//
// Function signatures are validated at runtime using reflection.
func (w *Workflow) NewTask(name string, fn any, options ...TaskOption) *Workflow {
	config := &taskConfig{}

	for _, opt := range options {
		opt(config)
	}

	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		panic("task function must be a function")
	}
	if fnType.NumIn() != 2 {
		panic("task function must have exactly 2 parameters: (ctx Context, input T)")
	}
	if fnType.NumOut() != 2 {
		panic("task function must return exactly 2 values: (output T, error)")
	}

	contextType := reflect.TypeOf((*Context)(nil)).Elem()
	durableContextType := reflect.TypeOf((*worker.DurableHatchetContext)(nil)).Elem()

	if config.isDurable {
		if !fnType.In(0).Implements(durableContextType) && fnType.In(0) != durableContextType {
			panic("first parameter for durable task must be DurableHatchetContext")
		}
	} else {
		if !fnType.In(0).Implements(contextType) && fnType.In(0) != contextType {
			panic("first parameter must be Context")
		}
	}

	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !fnType.Out(1).Implements(errorType) {
		panic("second return value must be error")
	}

	wrapper := func(ctx Context, input any) (any, error) {
		args := []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(input),
		}

		results := fnValue.Call(args)

		output := results[0].Interface()
		var err error
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}

		return output, err
	}

	taskOpts := create.WorkflowTask[any, any]{
		Name:                   name,
		Retries:                config.retries,
		RetryBackoffFactor:     config.retryBackoffFactor,
		RetryMaxBackoffSeconds: config.retryMaxBackoffSeconds,
		ExecutionTimeout:       config.executionTimeout,
		Concurrency:           config.concurrency,
		RateLimits:            config.rateLimits,
		Parents:               config.parents,
		WaitFor:               config.waitFor,
		SkipIf:                config.skipIf,
	}

	w.declaration.Task(taskOpts, wrapper)

	return w
}

// AddTask adds a task to the workflow and returns a Task reference for DAG building.
//
// The function parameter must have the signature:
//   func(ctx Context, input T) (T, error)
//
// For durable tasks, use:
//   func(ctx DurableContext, input T) (T, error)
//
// Function signatures are validated at runtime using reflection.
func (w *Workflow) AddTask(name string, fn any, options ...TaskOption) *Task {
	config := &taskConfig{}

	for _, opt := range options {
		opt(config)
	}

	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		panic("task function must be a function")
	}
	if fnType.NumIn() != 2 {
		panic("task function must have exactly 2 parameters: (ctx Context, input T)")
	}
	if fnType.NumOut() != 2 {
		panic("task function must return exactly 2 values: (output T, error)")
	}

	contextType := reflect.TypeOf((*Context)(nil)).Elem()
	durableContextType := reflect.TypeOf((*worker.DurableHatchetContext)(nil)).Elem()

	if config.isDurable {
		if !fnType.In(0).Implements(durableContextType) && fnType.In(0) != durableContextType {
			panic("first parameter for durable task must be DurableHatchetContext")
		}
	} else {
		if !fnType.In(0).Implements(contextType) && fnType.In(0) != contextType {
			panic("first parameter must be Context")
		}
	}

	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !fnType.Out(1).Implements(errorType) {
		panic("second return value must be error")
	}

	wrapper := func(ctx Context, input any) (any, error) {
		args := []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(input),
		}

		results := fnValue.Call(args)

		output := results[0].Interface()
		var err error
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}

		return output, err
	}

	taskOpts := create.WorkflowTask[any, any]{
		Name:                   name,
		Retries:                config.retries,
		RetryBackoffFactor:     config.retryBackoffFactor,
		RetryMaxBackoffSeconds: config.retryMaxBackoffSeconds,
		ExecutionTimeout:       config.executionTimeout,
		Concurrency:           config.concurrency,
		RateLimits:            config.rateLimits,
		Parents:               config.parents,
		WaitFor:               config.waitFor,
		SkipIf:                config.skipIf,
	}

	namedTask := w.declaration.Task(taskOpts, wrapper)

	return &Task{NamedTask: namedTask}
}

// NewDurableTask adds a durable task to the workflow.
// This is a convenience method that automatically sets the WithDurable option.
func (w *Workflow) NewDurableTask(name string, fn any, options ...TaskOption) *Workflow {
	durableOptions := append(options, WithDurable())
	return w.NewTask(name, fn, durableOptions...)
}

// NewStandaloneTask creates a workflow containing a single task.
// Workflow-level options (cron, events) are extracted from task options.
func NewStandaloneTask(name string, fn any, v1Client v1.HatchetClient, options ...TaskOption) *Workflow {
	config := &taskConfig{}

	for _, opt := range options {
		opt(config)
	}

	var workflowOptions []WorkflowOption
	if len(config.onCron) > 0 {
		workflowOptions = append(workflowOptions, WithWorkflowCron(config.onCron...))
	}
	if len(config.onEvents) > 0 {
		workflowOptions = append(workflowOptions, WithWorkflowEvents(config.onEvents...))
	}

	workflow := NewWorkflow(name, v1Client, workflowOptions...)

	taskOptions := make([]TaskOption, 0)
	for _, opt := range options {
		if isTaskLevelOption(opt) {
			taskOptions = append(taskOptions, opt)
		}
	}
	workflow.NewTask(name, fn, taskOptions...)

	return workflow
}

func isTaskLevelOption(opt TaskOption) bool {
	config := &taskConfig{}
	opt(config)

	return config.retries != 0 || config.retryBackoffFactor != 0 || config.retryMaxBackoffSeconds != 0 ||
		config.executionTimeout != 0 || len(config.concurrency) > 0 || config.isDurable
}

// OnFailure sets a failure handler for the workflow.
// The handler will be called when any task in the workflow fails.
func (w *Workflow) OnFailure(fn any) *Workflow {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		panic("onFailure function must be a function")
	}
	if fnType.NumIn() != 2 {
		panic("onFailure function must have exactly 2 parameters: (ctx Context, input T)")
	}
	if fnType.NumOut() != 2 {
		panic("onFailure function must return exactly 2 values: (output T, error)")
	}

	contextType := reflect.TypeOf((*Context)(nil)).Elem()
	if !fnType.In(0).Implements(contextType) && fnType.In(0) != contextType {
		panic("first parameter must be Context")
	}

	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !fnType.Out(1).Implements(errorType) {
		panic("second return value must be error")
	}

	wrapper := func(ctx Context, input any) (any, error) {
		args := []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(input),
		}

		results := fnValue.Call(args)

		output := results[0].Interface()
		var err error
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}

		return output, err
	}

	w.declaration.OnFailure(
		create.WorkflowOnFailureTask[any, any]{},
		wrapper,
	)

	return w
}

// Dump implements the WorkflowBase interface for internal use.
func (w *Workflow) Dump() (*contracts.CreateWorkflowVersionRequest, []workflow.NamedFunction, []workflow.NamedFunction, workflow.WrappedTaskFn) {
	return w.declaration.Dump()
}

// Workflow execution methods

// Run executes the workflow with the provided input and waits for completion.
func (w *Workflow) Run(ctx context.Context, input any) (any, error) {
	return w.declaration.Run(ctx, input)
}

// RunNoWait executes the workflow with the provided input without waiting for completion.
// Returns a workflow run reference that can be used to track the run status.
func (w *Workflow) RunNoWait(ctx context.Context, input any) (*v0Client.Workflow, error) {
	return w.declaration.RunNoWait(ctx, input)
}

// Cron schedules the workflow to run on a regular basis using a cron expression.
func (w *Workflow) Cron(ctx context.Context, name string, cronExpr string, input any) (*rest.CronWorkflows, error) {
	return w.declaration.Cron(ctx, name, cronExpr, input)
}

// Schedule schedules the workflow to run at a specific time.
func (w *Workflow) Schedule(ctx context.Context, triggerAt time.Time, input any) (*rest.ScheduledWorkflows, error) {
	return w.declaration.Schedule(ctx, triggerAt, input)
}
