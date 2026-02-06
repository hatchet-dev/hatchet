package hatchet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
	"github.com/hatchet-dev/hatchet/sdks/go/internal"
)

type RunPriority = features.RunPriority

type runOpts struct {
	AdditionalMetadata *map[string]string
	Priority           *RunPriority
	Sticky             *bool
	Key                *string
}

type RunOptFunc func(*runOpts)

// WithRunMetadata sets the additional metadata for the workflow run.
func WithRunMetadata(metadata map[string]string) RunOptFunc {
	return func(opts *runOpts) {
		opts.AdditionalMetadata = &metadata
	}
}

// WithRunPriority sets the priority for the workflow run.
func WithRunPriority(priority RunPriority) RunOptFunc {
	return func(opts *runOpts) {
		opts.Priority = &priority
	}
}

// WithRunSticky enables stickiness for the child workflow run.
func WithRunSticky(sticky bool) RunOptFunc {
	return func(opts *runOpts) {
		opts.Sticky = &sticky
	}
}

// WithRunKey sets the key for the child workflow run.
func WithRunKey(key string) RunOptFunc {
	return func(opts *runOpts) {
		opts.Key = &key
	}
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

	// Try to convert using JSON marshal/unmarshal
	if expectedType.Kind() == reflect.Struct {
		// Marshal the input to JSON
		jsonData, err := json.Marshal(input)
		if err != nil {
			// If marshaling fails, return the original input value
			return reflect.ValueOf(input)
		}

		// Create a new instance of the expected type
		result := reflect.New(expectedType)

		// Unmarshal JSON into the new instance
		err = json.Unmarshal(jsonData, result.Interface())
		if err != nil {
			panic(err)
		}

		// Return the dereferenced value (not the pointer)
		return result.Elem()
	}

	return reflect.ValueOf(input)
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
	taskDefaults    *create.TaskDefaults
	defaultPriority *RunPriority
	stickyStrategy  *types.StickyStrategy
	cronInput       *string
	version         string
	description     string
	onCron          []string
	onEvents        []string
	concurrency     []types.Concurrency
}

// WithWorkflowCron configures the workflow to run on a cron schedule.
// Multiple cron expressions can be provided.
func WithWorkflowCron(cronExpressions ...string) WorkflowOption {
	return func(config *workflowConfig) {
		config.onCron = cronExpressions
	}
}

// WithWorkflowCronInput sets the input for cron workflows.
func WithWorkflowCronInput(input any) WorkflowOption {
	return func(config *workflowConfig) {
		inputJSON := "{}"

		if input != nil {
			bytes, err := json.Marshal(input)
			if err != nil {
				panic(fmt.Errorf("could not marshal cron input: %w", err))
			}

			inputJSON = string(bytes)
		}

		config.cronInput = &inputJSON
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

// WithWorkflowDefaultPriority sets the default priority for the workflow.
func WithWorkflowDefaultPriority(priority RunPriority) WorkflowOption {
	return func(config *workflowConfig) {
		config.defaultPriority = &priority
	}
}

// WithWorkflowStickyStrategy sets the sticky strategy for the workflow.
func WithWorkflowStickyStrategy(stickyStrategy types.StickyStrategy) WorkflowOption {
	return func(config *workflowConfig) {
		config.stickyStrategy = &stickyStrategy
	}
}

// newWorkflow creates a new workflow definition.
func newWorkflow(name string, v0Client v0Client.Client, options ...WorkflowOption) *Workflow {
	config := &workflowConfig{}

	for _, opt := range options {
		opt(config)
	}

	if len(config.onCron) > 0 && config.cronInput == nil {
		emptyJSON := "{}"
		config.cronInput = &emptyJSON
	}

	createOpts := create.WorkflowCreateOpts[any]{
		Name:           name,
		Version:        config.version,
		Description:    config.description,
		OnEvents:       config.onEvents,
		OnCron:         config.onCron,
		CronInput:      config.cronInput,
		Concurrency:    config.concurrency,
		TaskDefaults:   config.taskDefaults,
		StickyStrategy: config.stickyStrategy,
	}

	if config.defaultPriority != nil {
		priority := int32(*config.defaultPriority)
		createOpts.DefaultPriority = &priority
	}

	declaration := internal.NewWorkflowDeclaration[any, any](createOpts, v0Client)

	return &Workflow{
		declaration: declaration,
		v0Client:    v0Client,
	}
}

// TaskOption configures a task instance.
type TaskOption func(*taskConfig)

type taskConfig struct {
	waitFor                condition.Condition
	skipIf                 condition.Condition
	description            string
	rateLimits             []*types.RateLimit
	onCron                 []string
	onEvents               []string
	defaultFilters         []types.DefaultFilter
	concurrency            []*types.Concurrency
	parents                []create.NamedTask
	scheduleTimeout        time.Duration
	executionTimeout       time.Duration
	retries                int32
	retryMaxBackoffSeconds int32
	retryBackoffFactor     float32
	isDurable              bool
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

// WithDescription sets a human-readable description for the task.
func WithDescription(description string) TaskOption {
	return func(config *taskConfig) {
		config.description = description
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

// Dump implements the WorkflowBase interface for internal use.
func (w *Workflow) Dump() (*v1.CreateWorkflowVersionRequest, []internal.NamedFunction, []internal.NamedFunction, internal.WrappedTaskFn) {
	return w.declaration.Dump()
}

// OnFailure sets a failure handler for the workflow.
// The handler will be called when any task in the workflow fails.
func (w *Workflow) OnFailure(fn any) {
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
}

// Workflow execution methods

// Run executes the workflow with the provided input and waits for completion.
func (w *Workflow) Run(ctx context.Context, input any, opts ...RunOptFunc) (*WorkflowResult, error) {
	workflowRunRef, err := w.RunNoWait(ctx, input, opts...)
	if err != nil {
		return nil, err
	}

	result, err := workflowRunRef.v0Workflow.Result()
	if err != nil {
		return nil, err
	}

	workflowResult, err := result.Results()
	if err != nil {
		return nil, err
	}

	return &WorkflowResult{result: workflowResult, RunId: workflowRunRef.RunId}, nil
}

// RunNoWait executes the workflow with the provided input without waiting for completion.
// Returns a workflow run reference that can be used to track the run status.
func (w *Workflow) RunNoWait(ctx context.Context, input any, opts ...RunOptFunc) (*WorkflowRunRef, error) {
	runOpts := &runOpts{}
	for _, opt := range opts {
		opt(runOpts)
	}

	var priority *int32
	if runOpts.Priority != nil {
		priority = &[]int32{int32(*runOpts.Priority)}[0]
	}

	var v0Opts []v0Client.RunOptFunc

	if runOpts.AdditionalMetadata != nil {
		v0Opts = append(v0Opts, v0Client.WithRunMetadata(*runOpts.AdditionalMetadata))
	}

	if priority != nil {
		v0Opts = append(v0Opts, v0Client.WithPriority(*priority))
	}

	var v0Workflow *v0Client.Workflow
	var err error

	hCtx, ok := ctx.(Context)
	if ok {
		v0Workflow, err = hCtx.SpawnWorkflow(w.declaration.Name(), input, &worker.SpawnWorkflowOpts{
			Key:                runOpts.Key,
			Sticky:             runOpts.Sticky,
			Priority:           priority,
			AdditionalMetadata: runOpts.AdditionalMetadata,
		})
	} else {
		v0Workflow, err = w.v0Client.Admin().RunWorkflow(w.declaration.Name(), input, v0Opts...)
	}

	if err != nil {
		return nil, err
	}

	return &WorkflowRunRef{RunId: v0Workflow.RunId(), v0Workflow: v0Workflow}, nil
}

// RunMany executes multiple workflow instances with different inputs.
func (w *Workflow) RunMany(ctx context.Context, inputs []RunManyOpt) ([]WorkflowRunRef, error) {
	var workflowRefs []WorkflowRunRef

	var wg sync.WaitGroup
	var errs []error
	var errsMutex sync.Mutex

	wg.Add(len(inputs))

	for _, input := range inputs {
		go func() {
			defer wg.Done()

			workflowRef, err := w.RunNoWait(ctx, input.Input, input.Opts...)
			if err != nil {
				errsMutex.Lock()
				errs = append(errs, err)
				errsMutex.Unlock()
				return
			}

			workflowRefs = append(workflowRefs, *workflowRef)
		}()
	}

	wg.Wait()

	return workflowRefs, errors.Join(errs...)
}
