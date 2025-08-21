package hatchet

import (
	"context"
	"reflect"
	"strings"
	"time"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
	"github.com/hatchet-dev/hatchet/sdks/go/internal"
)

// RunOpts is a type that represents the options for running a workflow.
type RunOpts struct {
	AdditionalMetadata *map[string]interface{}
	Priority           *int32
	// Sticky             *bool
	// Key                *string
}

type RunOptFunc = v0Client.RunOptFunc

func WithRunMetadata(metadata interface{}) RunOptFunc {
	return v0Client.WithRunMetadata(metadata)
}

// RunPriority is the priority for a workflow run.
type RunPriority int32

const (
	// RunPriorityLow is the lowest priority for a workflow run.
	RunPriorityLow RunPriority = 1
	// RunPriorityMedium is the medium priority for a workflow run.
	RunPriorityMedium RunPriority = 2
	// RunPriorityHigh is the highest priority for a workflow run.
	RunPriorityHigh RunPriority = 3
)

func WithPriority(priority RunPriority) RunOptFunc {
	return v0Client.WithPriority(int32(priority))
}

// convertInputToType converts input (typically map[string]interface{}) to the expected struct type
func convertInputToType(input any, expectedType reflect.Type) reflect.Value {
	if input == nil {
		return reflect.Zero(expectedType)
	}

	inputValue := reflect.ValueOf(input)
	if inputValue.Type().AssignableTo(expectedType) {
		return inputValue
	}

	// Try to convert map[string]any to the expected struct type
	if inputMap, ok := input.(map[string]any); ok && expectedType.Kind() == reflect.Struct {
		convertedInput := reflect.New(expectedType).Elem()
		for i := 0; i < expectedType.NumField(); i++ {
			field := expectedType.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" {
				jsonTag = field.Name
			} else {
				// Handle JSON tag options like "count,omitempty" -> "count"
				if commaIndex := strings.Index(jsonTag, ","); commaIndex != -1 {
					jsonTag = jsonTag[:commaIndex]
				}
			}
			if val, exists := inputMap[jsonTag]; exists {
				fieldValue := convertedInput.Field(i)
				if fieldValue.CanSet() {
					valReflect := reflect.ValueOf(val)
					if valReflect.Type().AssignableTo(field.Type) {
						fieldValue.Set(valReflect)
					} else {
						// Try to convert common type mismatches
						converted, ok := convertValue(val, field.Type)
						if ok {
							fieldValue.Set(converted)
						}
					}
				}
			}
		}
		return convertedInput
	}

	return reflect.ValueOf(input)
}

// convertValue attempts to convert a value to the target type for common type mismatches
func convertValue(val any, targetType reflect.Type) (reflect.Value, bool) {
	valReflect := reflect.ValueOf(val)

	// Handle numeric conversions (e.g., float64 -> int, int -> float64)
	if valReflect.Kind() == reflect.Float64 && targetType.Kind() == reflect.Int {
		if f64, ok := val.(float64); ok {
			return reflect.ValueOf(int(f64)), true
		}
	}
	if valReflect.Kind() == reflect.Float64 && targetType.Kind() == reflect.Int32 {
		if f64, ok := val.(float64); ok {
			return reflect.ValueOf(int32(f64)), true
		}
	}
	if valReflect.Kind() == reflect.Float64 && targetType.Kind() == reflect.Int64 {
		if f64, ok := val.(float64); ok {
			return reflect.ValueOf(int64(f64)), true
		}
	}

	return reflect.Value{}, false
}

// Workflow defines a Hatchet workflow, which can then declare tasks and be run, scheduled, and so on.
type Workflow struct {
	declaration internal.WorkflowDeclaration[any, any]
	v0Client    v0Client.Client
}

// GetName returns the resolved workflow name (including namespace if applicable).
func (w *Workflow) GetName() string {
	return w.declaration.Name()
}

// WorkflowOption configures a workflow instance.
type WorkflowOption func(*workflowConfig)

type workflowConfig struct {
	onCron       []string
	onEvents     []string
	concurrency  []types.Concurrency
	version      string
	description  string
	taskDefaults *create.TaskDefaults
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

// WithWorkflowConcurrency sets concurrency controls for the workflow.
func WithWorkflowConcurrency(concurrency ...types.Concurrency) WorkflowOption {
	return func(config *workflowConfig) {
		config.concurrency = concurrency
	}
}

// WithWorkflowTaskDefaults sets the default configuration for all tasks in the workflow.
func WithWorkflowTaskDefaults(defaults *create.TaskDefaults) WorkflowOption {
	return func(config *workflowConfig) {
		config.taskDefaults = defaults
	}
}

// newWorkflow creates a new workflow definition.
func newWorkflow(name string, v0Client v0Client.Client, options ...WorkflowOption) *Workflow {
	config := &workflowConfig{}

	for _, opt := range options {
		opt(config)
	}

	declaration := internal.NewWorkflowDeclaration[any, any](
		create.WorkflowCreateOpts[any]{
			Name:         name,
			Version:      config.version,
			Description:  config.description,
			OnEvents:     config.onEvents,
			OnCron:       config.onCron,
			Concurrency:  config.concurrency,
			TaskDefaults: config.taskDefaults,
		},
		v0Client,
	)

	return &Workflow{
		declaration: declaration,
		v0Client:    v0Client,
	}
}

// TaskOption configures a task instance.
type TaskOption func(*taskConfig)

type taskConfig struct {
	retries                int32
	retryBackoffFactor     float32
	retryMaxBackoffSeconds int32
	executionTimeout       time.Duration
	scheduleTimeout        time.Duration
	onCron                 []string
	onEvents               []string
	defaultFilters         []types.DefaultFilter
	concurrency            []*types.Concurrency
	rateLimits             []*types.RateLimit
	isDurable              bool
	parents                []create.NamedTask
	waitFor                condition.Condition
	skipIf                 condition.Condition
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

// WithScheduleTimeout sets the maximum time a task can wait to be scheduled.
func WithScheduleTimeout(timeout time.Duration) TaskOption {
	return func(config *taskConfig) {
		config.scheduleTimeout = timeout
	}
}

// WithExecutionTimeout sets the maximum execution duration for a task.
func WithExecutionTimeout(timeout time.Duration) TaskOption {
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

// withDurable marks a task as durable, enabling persistent state and long-running operations.
func withDurable() TaskOption {
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
func WithParents(parents ...*Task) TaskOption {
	return func(config *taskConfig) {
		// Convert *Task to create.NamedTask
		namedTasks := make([]create.NamedTask, len(parents))
		for i, parent := range parents {
			namedTasks[i] = parent
		}
		config.parents = namedTasks
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

// Task represents a task reference for building DAGs and conditions.
type Task struct {
	name string
}

// Name returns the name of the task.
func (t *Task) GetName() string {
	return t.name
}

// NewTask transforms a function into a Hatchet task that runs as part of a workflow.
//
// The function parameter must have the signature:
//
//	func(ctx hatchet.Context, input any) (any, error)
//
// Function signatures are validated at runtime using reflection.
func (w *Workflow) NewTask(name string, fn any, options ...TaskOption) *Task {
	if name == "" {
		panic("task name cannot be empty")
	}

	if fn == nil {
		panic("task '" + name + "' has a nil input function")
	}

	config := &taskConfig{}

	for _, opt := range options {
		opt(config)
	}

	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		panic("fn must be a function")
	}

	if fnType.NumIn() != 2 {
		panic("fn must have exactly 2 parameters: (ctx hatchet.Context, input T)")
	}

	if fnType.NumOut() != 2 {
		panic("fn must return exactly 2 values: (output T, err error)")
	}

	contextType := reflect.TypeOf((*Context)(nil)).Elem()
	durableContextType := reflect.TypeOf((*worker.DurableHatchetContext)(nil)).Elem()

	if config.isDurable {
		if !fnType.In(0).Implements(durableContextType) && fnType.In(0) != durableContextType {
			panic("first parameter for durable task must be hatchet.DurableContext")
		}
	} else {
		if !fnType.In(0).Implements(contextType) && fnType.In(0) != contextType {
			panic("first parameter must be hatchet.Context")
		}
	}

	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !fnType.Out(1).Implements(errorType) {
		panic("second return value must be error")
	}

	wrapper := func(ctx Context, input any) (any, error) {
		// Convert the input to the expected type
		expectedInputType := fnType.In(1)
		convertedInput := convertInputToType(input, expectedInputType)

		// For durable tasks, we need to pass the context as the expected type
		var contextArg reflect.Value
		durableContextType := reflect.TypeOf((*worker.DurableHatchetContext)(nil)).Elem()
		if fnType.In(0).Implements(durableContextType) || fnType.In(0) == durableContextType {
			// For durable tasks, convert the context to DurableHatchetContext
			durableCtx := worker.NewDurableHatchetContext(ctx)
			contextArg = reflect.ValueOf(durableCtx)
		} else {
			contextArg = reflect.ValueOf(ctx)
		}

		args := []reflect.Value{
			contextArg,
			convertedInput,
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
		ScheduleTimeout:        config.scheduleTimeout,
		Concurrency:            config.concurrency,
		RateLimits:             config.rateLimits,
		Parents:                config.parents,
		WaitFor:                config.waitFor,
		SkipIf:                 config.skipIf,
	}

	w.declaration.Task(taskOpts, wrapper)

	return &Task{name: name}
}

// NewDurableTask transforms a function into a durable Hatchet task that runs as part of a workflow.
//
// The function parameter must have the signature:
//
//	func(ctx hatchet.DurableContext, input any) (any, error)
//
// Function signatures are validated at runtime using reflection.
func (w *Workflow) NewDurableTask(name string, fn any, options ...TaskOption) *Task {
	durableOptions := append(options, withDurable())
	return w.NewTask(name, fn, durableOptions...)
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
		// Convert the input to the expected type
		expectedInputType := fnType.In(1)
		convertedInput := convertInputToType(input, expectedInputType)

		// For durable tasks, we need to pass the context as the expected type
		var contextArg reflect.Value
		durableContextType := reflect.TypeOf((*worker.DurableHatchetContext)(nil)).Elem()
		if fnType.In(0).Implements(durableContextType) || fnType.In(0) == durableContextType {
			// For durable tasks, convert the context to DurableHatchetContext
			durableCtx := worker.NewDurableHatchetContext(ctx)
			contextArg = reflect.ValueOf(durableCtx)
		} else {
			contextArg = reflect.ValueOf(ctx)
		}

		args := []reflect.Value{
			contextArg,
			convertedInput,
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
func (w *Workflow) Dump() (*contracts.CreateWorkflowVersionRequest, []internal.NamedFunction, []internal.NamedFunction, internal.WrappedTaskFn) {
	return w.declaration.Dump()
}

// Workflow execution methods

// Run executes the workflow with the provided input and waits for completion.
func (w *Workflow) Run(ctx context.Context, input any) (any, error) {
	return w.declaration.Run(ctx, input)
}

// RunNoWait executes the workflow with the provided input without waiting for completion.
// Returns a workflow run reference that can be used to track the run status.
func (w *Workflow) RunNoWait(ctx context.Context, input any) (*WorkflowRef, error) {
	wf, err := w.declaration.RunNoWait(ctx, input)
	if err != nil {
		return nil, err
	}

	return &WorkflowRef{RunId: wf.RunId()}, nil
}

// RunAsChildOpts is the options for running a workflow as a child workflow.
type RunAsChildOpts = internal.RunAsChildOpts

// RunAsChild executes the workflow as a child workflow with the provided input.
func (w *Workflow) RunAsChild(ctx worker.HatchetContext, input any, opts RunAsChildOpts) (any, error) {
	return w.declaration.RunAsChild(ctx, input, opts)
}
