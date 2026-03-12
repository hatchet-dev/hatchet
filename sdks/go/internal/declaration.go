// Package internal provides internal functionality for the Hatchet Go SDK
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
	"github.com/hatchet-dev/hatchet/sdks/go/internal/task"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// WrappedTaskFn represents a task function that can be executed by the Hatchet worker.
// It takes a HatchetContext and returns an interface{} result and an error.
type WrappedTaskFn func(ctx worker.HatchetContext) (interface{}, error)

// DurableWrappedTaskFn represents a durable task function that can be executed by the Hatchet worker.
// It takes a DurableHatchetContext and returns an interface{} result and an error.
type DurableWrappedTaskFn func(ctx worker.DurableHatchetContext) (interface{}, error)

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

type RunOpts struct {
	AdditionalMetadata *map[string]interface{}
	Priority           *int32
}

type RunAsChildOpts struct {
	RunOpts
	Sticky *bool
	Key    *string
}

// WorkflowDeclaration represents a workflow with input type I and output type O.
// It provides methods to define tasks, specify dependencies, and execute the workflow.
type WorkflowDeclaration[I, O any] interface {
	WorkflowBase

	// Name returns the resolved workflow name (including namespace if applicable).
	Name() string

	// Task registers a task that will be executed as part of the workflow
	Task(opts create.WorkflowTask[I, O], fn func(ctx worker.HatchetContext, input I) (interface{}, error)) *task.TaskDeclaration[I]

	// DurableTask registers a durable task that will be executed as part of the workflow.
	// Durable tasks can be paused and resumed across workflow runs, making them suitable
	// for long-running operations or tasks that require human intervention.
	DurableTask(opts create.WorkflowTask[I, O], fn func(ctx worker.DurableHatchetContext, input I) (interface{}, error)) *task.DurableTaskDeclaration[I]

	// OnFailureTask registers a task that will be executed if the workflow fails.
	OnFailure(opts create.WorkflowOnFailureTask[I, O], fn func(ctx worker.HatchetContext, input I) (interface{}, error)) *task.OnFailureTaskDeclaration[I]

	// Cron schedules the workflow to run on a regular basis using a cron expression.
	Cron(ctx context.Context, name string, cronExpr string, input I, opts ...v0Client.RunOptFunc) (*rest.CronWorkflows, error)

	// Schedule schedules the workflow to run at a specific time.
	Schedule(ctx context.Context, triggerAt time.Time, input I, opts ...v0Client.RunOptFunc) (*rest.ScheduledWorkflows, error)

	// Get retrieves the current state of the workflow.
	Get(ctx context.Context) (*rest.Workflow, error)

	// // IsPaused checks if the workflow is currently paused.
	// IsPaused(ctx context.Context) (bool, error)

	// // Pause pauses the assignment of new workflow runs.
	// Pause(ctx context.Context) error

	// // Unpause resumes the assignment of workflow runs.
	// Unpause(ctx context.Context) error

	// Metrics retrieves metrics for this workflow.
	Metrics(ctx context.Context, opts ...rest.WorkflowGetMetricsParams) (*rest.WorkflowMetrics, error)

	// QueueMetrics retrieves queue metrics for this workflow.
	QueueMetrics(ctx context.Context, opts ...rest.TenantGetQueueMetricsParams) (*rest.TenantQueueMetrics, error)
}

// Define a TaskDeclaration with specific output type
type TaskWithSpecificOutput[I any, T any] struct {
	Name string
	Fn   func(ctx worker.HatchetContext, input I) (*T, error)
}

// workflowDeclarationImpl is the concrete implementation of WorkflowDeclaration.
// It contains all the data and logic needed to define and execute a workflow.
type workflowDeclarationImpl[I any, O any] struct {
	v0        v0Client.Client
	crons     *features.CronsClient
	schedules *features.SchedulesClient
	metrics   *features.MetricsClient
	workflows *features.WorkflowsClient

	outputKey *string

	name           string
	Version        *string
	Description    *string
	OnEvents       []string
	OnCron         []string
	CronInput      *string
	Concurrency    []types.Concurrency
	OnFailureTask  *task.OnFailureTaskDeclaration[I]
	StickyStrategy *types.StickyStrategy

	TaskDefaults *create.TaskDefaults

	tasks        []*task.TaskDeclaration[I]
	durableTasks []*task.DurableTaskDeclaration[I]

	// Store task functions with their specific output types
	taskFuncs        map[string]interface{}
	durableTaskFuncs map[string]interface{}

	// Map to store task output setters
	outputSetters map[string]func(*O, interface{})

	DefaultPriority *int32
	DefaultFilters  []types.DefaultFilter
}

// NewWorkflowDeclaration creates a new workflow declaration with the specified options and client.
// The workflow will have input type I and output type O.
func NewWorkflowDeclaration[I any, O any](opts create.WorkflowCreateOpts[I], v0 v0Client.Client) WorkflowDeclaration[I, O] {
	api := v0.API()
	tenantId := v0.TenantId()
	namespace := v0.Namespace()

	crons := features.NewCronsClient(api, tenantId)
	schedules := features.NewSchedulesClient(api, tenantId, &namespace)
	metrics := features.NewMetricsClient(api, tenantId)
	workflows := features.NewWorkflowsClient(api, tenantId)

	workflowName := opts.Name

	ns := v0.Namespace()

	if ns != "" && !strings.HasPrefix(opts.Name, ns) {
		workflowName = fmt.Sprintf("%s%s", ns, workflowName)
	}

	onEvents := opts.OnEvents

	if ns != "" && len(onEvents) > 0 {
		// Prefix the events with the namespace
		onEvents = make([]string, len(opts.OnEvents))

		for i, event := range opts.OnEvents {
			onEvents[i] = fmt.Sprintf("%s%s", ns, event)
		}
	}

	wf := &workflowDeclarationImpl[I, O]{
		v0:          v0,
		crons:       crons,
		schedules:   schedules,
		metrics:     metrics,
		workflows:   workflows,
		name:        workflowName,
		OnEvents:    onEvents,
		OnCron:      opts.OnCron,
		CronInput:   opts.CronInput,
		Concurrency: opts.Concurrency,
		// OnFailureTask:    opts.OnFailureTask, // TODO: add this back in
		StickyStrategy:   opts.StickyStrategy,
		TaskDefaults:     opts.TaskDefaults,
		outputKey:        opts.OutputKey,
		tasks:            []*task.TaskDeclaration[I]{},
		taskFuncs:        make(map[string]interface{}),
		durableTasks:     []*task.DurableTaskDeclaration[I]{},
		durableTaskFuncs: make(map[string]interface{}),
		outputSetters:    make(map[string]func(*O, interface{})),
		DefaultPriority:  opts.DefaultPriority,
		DefaultFilters:   opts.DefaultFilters,
	}

	if opts.Version != "" {
		wf.Version = &opts.Version
	}

	if opts.Description != "" {
		wf.Description = &opts.Description
	}

	return wf
}

// Name returns the resolved workflow name (including namespace if applicable).
func (w *workflowDeclarationImpl[I, O]) Name() string {
	return w.name
}

// Task registers a standard (non-durable) task with the workflow
func (w *workflowDeclarationImpl[I, O]) Task(opts create.WorkflowTask[I, O], fn func(ctx worker.HatchetContext, input I) (interface{}, error)) *task.TaskDeclaration[I] {
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

		// If the field isn't found by name, try to find it by JSON tag
		resultType := resultValue.Type()
		for i := 0; i < resultType.NumField(); i++ {
			fieldType := resultType.Field(i)
			jsonTag := fieldType.Tag.Get("json")
			// Extract the name part from the json tag (before any comma)
			if commaIdx := strings.Index(jsonTag, ","); commaIdx > 0 {
				jsonTag = jsonTag[:commaIdx]
			}
			if jsonTag == name || strings.EqualFold(fieldType.Name, name) {
				field = resultValue.Field(i)
				break
			}
		}

		if field.IsValid() && field.CanSet() {
			outputValue := reflect.ValueOf(output).Elem()
			field.Set(outputValue)
		}
	}

	// Create a generic task function that wraps the specific one
	genericFn := func(ctx worker.HatchetContext, input I) (*any, error) {
		// Use reflection to call the specific function
		fnValue := reflect.ValueOf(fn)
		inputs := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(input)}
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
		parentNames[i] = parent.GetName()
	}

	taskDecl := &task.TaskDeclaration[I]{
		Name:     opts.Name,
		Fn:       genericFn,
		Parents:  parentNames,
		WaitFor:  opts.WaitFor,
		SkipIf:   opts.SkipIf,
		CancelIf: opts.CancelIf,
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
func (w *workflowDeclarationImpl[I, O]) DurableTask(opts create.WorkflowTask[I, O], fn func(ctx worker.DurableHatchetContext, input I) (interface{}, error)) *task.DurableTaskDeclaration[I] {
	name := opts.Name

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
	genericFn := func(ctx worker.DurableHatchetContext, input I) (*any, error) {
		// Use reflection to call the specific function
		fnValue := reflect.ValueOf(fn)
		inputs := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(input)}
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
		parentNames[i] = parent.GetName()
	}

	labels := make(map[string]*types.DesiredWorkerLabel)

	for k, v := range opts.WorkerLabels {
		labels[k] = &types.DesiredWorkerLabel{
			Value:      fmt.Sprintf("%v-durable", v.Value),
			Required:   v.Required,
			Weight:     v.Weight,
			Comparator: v.Comparator,
		}
	}

	taskDecl := &task.DurableTaskDeclaration[I]{
		Name:     opts.Name,
		Fn:       genericFn,
		Parents:  parentNames,
		WaitFor:  opts.WaitFor,
		SkipIf:   opts.SkipIf,
		CancelIf: opts.CancelIf,
		TaskShared: task.TaskShared{
			ExecutionTimeout:       executionTimeout,
			ScheduleTimeout:        scheduleTimeout,
			Retries:                retries,
			RetryBackoffFactor:     retryBackoffFactor,
			RetryMaxBackoffSeconds: retryMaxBackoffSeconds,
			RateLimits:             opts.RateLimits,
			WorkerLabels:           labels,
			Concurrency:            opts.Concurrency,
		},
	}

	w.durableTasks = append(w.durableTasks, taskDecl)
	w.durableTaskFuncs[name] = fn

	return taskDecl
}

// OnFailureTask registers a task that will be executed if the workflow fails.
func (w *workflowDeclarationImpl[I, O]) OnFailure(opts create.WorkflowOnFailureTask[I, O], fn func(ctx worker.HatchetContext, input I) (interface{}, error)) *task.OnFailureTaskDeclaration[I] {
	// Use reflection to validate the function type
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func ||
		fnType.NumIn() != 2 ||
		fnType.NumOut() != 2 ||
		!fnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic("Invalid function type for on failure task: must be func(I, worker.HatchetContext) (*T, error)")
	}

	// Create a generic task function that wraps the specific one
	genericFn := func(ctx worker.HatchetContext, input I) (*any, error) {
		// Use reflection to call the specific function
		fnValue := reflect.ValueOf(fn)
		inputs := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(input)}
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

	taskDecl := &task.OnFailureTaskDeclaration[I]{
		Fn: genericFn,
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

	w.OnFailureTask = taskDecl

	return taskDecl
}

// Cron schedules the workflow to run on a regular basis using a cron expression.
func (w *workflowDeclarationImpl[I, O]) Cron(ctx context.Context, name string, cronExpr string, input I, opts ...v0Client.RunOptFunc) (*rest.CronWorkflows, error) {
	var inputMap map[string]interface{}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(inputBytes, &inputMap); err != nil {
		return nil, err
	}

	cronTriggerOpts := features.CreateCronTrigger{
		Name:       name,
		Expression: cronExpr,
		Input:      inputMap,
	}

	runOpts := &admincontracts.TriggerWorkflowRequest{}

	for _, opt := range opts {
		opt(runOpts)
	}

	if runOpts.Priority != nil {
		cronTriggerOpts.Priority = &[]features.RunPriority{features.RunPriority(*runOpts.Priority)}[0]
	}

	if runOpts.AdditionalMetadata != nil {
		additionalMeta := make(map[string]interface{})
		json.Unmarshal([]byte(*runOpts.AdditionalMetadata), &additionalMeta)
		cronTriggerOpts.AdditionalMetadata = additionalMeta
	}

	cronWorkflow, err := w.crons.Create(ctx, w.name, cronTriggerOpts)
	if err != nil {
		return nil, err
	}

	return cronWorkflow, nil
}

// Schedule schedules the workflow to run at a specific time.
func (w *workflowDeclarationImpl[I, O]) Schedule(ctx context.Context, triggerAt time.Time, input I, opts ...v0Client.RunOptFunc) (*rest.ScheduledWorkflows, error) {
	var inputMap map[string]interface{}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(inputBytes, &inputMap); err != nil {
		return nil, err
	}

	triggerOpts := features.CreateScheduledRunTrigger{
		TriggerAt: triggerAt,
		Input:     inputMap,
	}

	runOpts := &admincontracts.TriggerWorkflowRequest{}

	for _, opt := range opts {
		opt(runOpts)
	}

	if runOpts.AdditionalMetadata != nil {
		additionalMetadata := make(map[string]interface{})
		json.Unmarshal([]byte(*runOpts.AdditionalMetadata), &additionalMetadata)

		triggerOpts.AdditionalMetadata = additionalMetadata
	}

	if runOpts.Priority != nil {
		priority := features.RunPriority(*runOpts.Priority)
		triggerOpts.Priority = &priority
	}

	scheduledWorkflow, err := w.schedules.Create(ctx, w.name, triggerOpts)
	if err != nil {
		return nil, err
	}

	return scheduledWorkflow, nil
}

// Dump converts the workflow declaration into a protobuf request and function mappings.
// This is used to serialize the workflow for transmission to the Hatchet server.
// Returns the workflow definition as a protobuf request, the task functions, and the on-failure task function.
func (w *workflowDeclarationImpl[I, O]) Dump() (*contracts.CreateWorkflowVersionRequest, []NamedFunction, []NamedFunction, WrappedTaskFn) {
	taskOpts := make([]*contracts.CreateTaskOpts, len(w.tasks))
	for i, task := range w.tasks {
		taskOpts[i] = task.Dump(w.name, w.TaskDefaults)
	}

	durableOpts := make([]*contracts.CreateTaskOpts, len(w.durableTasks))
	for i, task := range w.durableTasks {
		durableOpts[i] = task.Dump(w.name, w.TaskDefaults)
	}

	tasksToRegister := append(taskOpts, durableOpts...)

	req := &contracts.CreateWorkflowVersionRequest{
		Tasks:           tasksToRegister,
		Name:            w.name,
		EventTriggers:   w.OnEvents,
		CronTriggers:    w.OnCron,
		CronInput:       w.CronInput,
		DefaultPriority: w.DefaultPriority,
	}

	if w.Version != nil {
		req.Version = *w.Version
	}

	if w.Description != nil {
		req.Description = *w.Description
	}

	for _, concurrency := range w.Concurrency {
		c := contracts.Concurrency{
			Expression: concurrency.Expression,
			MaxRuns:    concurrency.MaxRuns,
		}

		if concurrency.LimitStrategy != nil {
			strategy := *concurrency.LimitStrategy
			strategyInt := contracts.ConcurrencyLimitStrategy_value[string(strategy)]
			strategyEnum := contracts.ConcurrencyLimitStrategy(strategyInt)
			c.LimitStrategy = &strategyEnum
		}

		req.ConcurrencyArr = append(req.ConcurrencyArr, &c)
	}

	if w.OnFailureTask != nil {
		req.OnFailureTask = w.OnFailureTask.Dump(w.name, w.TaskDefaults)
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
				inputVal := reflect.ValueOf(input)
				if !inputVal.IsValid() {
					// input is a nil interface (e.g. I = any with null input),
					// use a zero value of the function's expected parameter type
					inputVal = reflect.Zero(fnValue.Type().In(1))
				}
				inputs := []reflect.Value{reflect.ValueOf(ctx), inputVal}
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
				inputVal := reflect.ValueOf(input)
				if !inputVal.IsValid() {
					inputVal = reflect.Zero(fnValue.Type().In(1))
				}
				inputs := []reflect.Value{reflect.ValueOf(durableCtx), inputVal}
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
			inputs := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(input)}
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

// Get retrieves the current state of the workflow.
func (w *workflowDeclarationImpl[I, O]) Get(ctx context.Context) (*rest.Workflow, error) {
	workflow, err := w.workflows.Get(ctx, w.name)
	if err != nil {
		return nil, err
	}

	return workflow, nil
}

// Metrics retrieves metrics for this workflow.
func (w *workflowDeclarationImpl[I, O]) Metrics(ctx context.Context, opts ...rest.WorkflowGetMetricsParams) (*rest.WorkflowMetrics, error) {
	var options rest.WorkflowGetMetricsParams
	if len(opts) > 0 {
		options = opts[0]
	}

	metrics, err := w.metrics.GetWorkflowMetrics(ctx, w.name, &options)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

// QueueMetrics retrieves queue metrics for this workflow.
func (w *workflowDeclarationImpl[I, O]) QueueMetrics(ctx context.Context, opts ...rest.TenantGetQueueMetricsParams) (*rest.TenantQueueMetrics, error) {
	var options rest.TenantGetQueueMetricsParams
	if len(opts) > 0 {
		options = opts[0]
	}

	// Ensure the workflow name is set
	if options.Workflows == nil {
		options.Workflows = &[]string{w.name}
	} else {
		// Add this workflow to the list if not already present
		found := false
		for _, wf := range *options.Workflows {
			if wf == w.name {
				found = true
				break
			}
		}
		if !found {
			*options.Workflows = append(*options.Workflows, w.name)
		}
	}

	metrics, err := w.metrics.GetQueueMetrics(ctx, &options)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}
