package hatchet

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck // SA1019: internal usage, same pattern as client.go
	pkgWorker "github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/sdks/go/internal"
)

// BatchTaskOption configures a standalone batch task.
type BatchTaskOption func(*batchTaskConfig)

type batchTaskConfig struct {
	batchMaxIntervalMs *int32
	batchGroupKey      *string
	batchGroupMaxRuns  *int32
	executionTimeout   time.Duration
	scheduleTimeout    time.Duration
	batchMaxSize       int32
	retries            int32
}

// WithBatchMaxSize sets the maximum number of buffered items before the batch flushes.
func WithBatchMaxSize(n int) BatchTaskOption {
	return func(c *batchTaskConfig) { c.batchMaxSize = int32(n) } // nolint: gosec
}

// WithBatchMaxInterval sets the maximum duration to wait before flushing an incomplete batch.
func WithBatchMaxInterval(d time.Duration) BatchTaskOption {
	return func(c *batchTaskConfig) {
		ms := int32(d.Milliseconds()) // nolint: gosec
		c.batchMaxIntervalMs = &ms
	}
}

// WithBatchGroupKey sets a CEL expression that partitions items into independent sub-batches.
// Example: "input.tenantId" groups items by their tenantId field.
func WithBatchGroupKey(key string) BatchTaskOption {
	return func(c *batchTaskConfig) { c.batchGroupKey = &key }
}

// WithBatchGroupMaxRuns limits the number of concurrently executing batches per group key.
func WithBatchGroupMaxRuns(n int) BatchTaskOption {
	return func(c *batchTaskConfig) {
		v := int32(n) // nolint: gosec
		c.batchGroupMaxRuns = &v
	}
}

// WithBatchRetries sets the retry count for batch task step runs.
func WithBatchRetries(n int) BatchTaskOption {
	return func(c *batchTaskConfig) { c.retries = int32(n) } // nolint: gosec
}

// WithBatchExecutionTimeout sets the maximum execution duration for the batch task.
func WithBatchExecutionTimeout(d time.Duration) BatchTaskOption {
	return func(c *batchTaskConfig) { c.executionTimeout = d }
}

// StandaloneBatchTask is a task that buffers individual invocations server-side and
// calls the batch fn once with all buffered items. From the caller's perspective,
// each .Run() call submits one item and blocks until the batch flushes.
type StandaloneBatchTask struct {
	v0Client v0Client.Client //nolint:staticcheck // SA1019: internal usage, same pattern as client.go
	req      *v1.CreateWorkflowVersionRequest
	batchFn  pkgWorker.BatchWrappedFn
	name     string
	taskName string
}

// GetName returns the workflow name (including namespace if applicable).
func (st *StandaloneBatchTask) GetName() string {
	return st.name
}

// Dump implements WorkflowBase so StandaloneBatchTask can be passed to WithWorkflows.
func (st *StandaloneBatchTask) Dump() (*v1.CreateWorkflowVersionRequest, []internal.NamedFunction, []internal.NamedFunction, internal.WrappedTaskFn) {
	actionID := strings.ToLower(fmt.Sprintf("%s:%s", st.name, st.taskName))
	namedFns := []internal.NamedFunction{
		{
			ActionID: actionID,
			BatchFn:  st.batchFn,
		},
	}
	return st.req, namedFns, nil, nil
}

// OnFailure is a no-op for batch tasks (on-failure handlers are not supported for batch tasks).
func (st *StandaloneBatchTask) OnFailure(_ any) {}

// Run submits one item to the batch and waits for its result.
func (st *StandaloneBatchTask) Run(ctx context.Context, input any, opts ...RunOptFunc) (*TaskResult, error) {
	runOpts := &runOpts{}
	for _, opt := range opts {
		opt(runOpts)
	}

	var v0Opts []v0Client.RunOptFunc
	if runOpts.AdditionalMetadata != nil {
		v0Opts = append(v0Opts, v0Client.WithRunMetadata(*runOpts.AdditionalMetadata))
	}
	if runOpts.Priority != nil {
		v0Opts = append(v0Opts, v0Client.WithPriority(int32(*runOpts.Priority)))
	}

	v0Workflow, err := st.v0Client.Admin().RunWorkflow(st.name, input, v0Opts...)
	if err != nil {
		return nil, err
	}

	result, err := v0Workflow.Result()
	if err != nil {
		return nil, err
	}

	results, err := result.Results()
	if err != nil {
		return nil, err
	}

	wr := &WorkflowResult{result: results, RunId: v0Workflow.RunId()}
	return wr.TaskOutput(st.taskName), nil
}

// RunNoWait submits one item without waiting for completion.
func (st *StandaloneBatchTask) RunNoWait(ctx context.Context, input any, opts ...RunOptFunc) (*WorkflowRunRef, error) {
	runOpts := &runOpts{}
	for _, opt := range opts {
		opt(runOpts)
	}

	var v0Opts []v0Client.RunOptFunc
	if runOpts.AdditionalMetadata != nil {
		v0Opts = append(v0Opts, v0Client.WithRunMetadata(*runOpts.AdditionalMetadata))
	}
	if runOpts.Priority != nil {
		v0Opts = append(v0Opts, v0Client.WithPriority(int32(*runOpts.Priority)))
	}

	v0Workflow, err := st.v0Client.Admin().RunWorkflow(st.name, input, v0Opts...)
	if err != nil {
		return nil, err
	}

	return &WorkflowRunRef{RunId: v0Workflow.RunId(), v0Workflow: v0Workflow}, nil
}

// RunMany submits multiple items without waiting.
func (st *StandaloneBatchTask) RunMany(ctx context.Context, inputs []RunManyOpt) ([]WorkflowRunRef, error) {
	refs := make([]WorkflowRunRef, 0, len(inputs))
	for _, inp := range inputs {
		ref, err := st.RunNoWait(ctx, inp.Input, inp.Opts...)
		if err != nil {
			return refs, err
		}
		refs = append(refs, *ref)
	}
	return refs, nil
}

// NewStandaloneBatchTask creates a standalone batch task.
//
// fn must have one of the following signatures:
//
//	func(ctx hatchet.Context, items []I) ([]O, error)
//	func(items []I) ([]O, error)
//
// where I is the input type for each item and O is the output type.
// Results are returned in the same order as the input items.
func (c *Client) NewStandaloneBatchTask(name string, fn any, opts ...BatchTaskOption) *StandaloneBatchTask {
	if name == "" {
		panic("batch task name cannot be empty")
	}
	if fn == nil {
		panic("batch task '" + name + "' has a nil function")
	}

	config := &batchTaskConfig{
		batchMaxSize: 10,
	}
	for _, opt := range opts {
		opt(config)
	}

	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		panic("batch task fn must be a function")
	}

	// Accept: func(ctx Context, items []I) ([]O, error)  — 2 in, 2 out
	//         func(items []I) ([]O, error)               — 1 in, 2 out
	hasCtxParam := false
	var itemSliceType reflect.Type

	errorType := reflect.TypeOf((*error)(nil)).Elem()
	contextType := reflect.TypeOf((*Context)(nil)).Elem()

	switch fnType.NumIn() {
	case 1:
		itemSliceType = fnType.In(0)
	case 2:
		if !fnType.In(0).Implements(contextType) {
			panic("batch task fn with 2 params: first param must be hatchet.Context")
		}
		hasCtxParam = true
		itemSliceType = fnType.In(1)
	default:
		panic("batch task fn must have 1 or 2 parameters")
	}

	if itemSliceType.Kind() != reflect.Slice {
		panic("batch task fn items parameter must be a slice")
	}
	inputElemType := itemSliceType.Elem()

	if fnType.NumOut() != 2 {
		panic("batch task fn must return exactly 2 values: ([]O, error)")
	}
	outputSliceType := fnType.Out(0)
	if outputSliceType.Kind() != reflect.Slice {
		panic("batch task fn first return value must be a slice")
	}
	if !fnType.Out(1).Implements(errorType) {
		panic("batch task fn second return value must be error")
	}

	// Build the namespace-aware workflow name.
	ns := c.legacyClient.Namespace()
	workflowName := name
	if ns != "" && !strings.HasPrefix(name, ns) {
		workflowName = ns + name
	}

	// Build the proto request.
	taskOpts := &v1.CreateTaskOpts{
		ReadableId: workflowName,
		Action:     strings.ToLower(fmt.Sprintf("%s:%s", workflowName, workflowName)),
		Retries:    config.retries,
		Batch: &v1.TaskBatchConfig{
			BatchMaxSize: config.batchMaxSize,
		},
	}

	if config.batchMaxIntervalMs != nil {
		taskOpts.Batch.BatchMaxInterval = config.batchMaxIntervalMs
	}
	if config.batchGroupKey != nil {
		taskOpts.Batch.BatchGroupKey = config.batchGroupKey
	}
	if config.batchGroupMaxRuns != nil {
		taskOpts.Batch.BatchGroupMaxRuns = config.batchGroupMaxRuns
	}
	if config.executionTimeout != 0 {
		taskOpts.Timeout = fmt.Sprintf("%ds", int(config.executionTimeout.Seconds()))
	}
	if config.scheduleTimeout != 0 {
		s := fmt.Sprintf("%ds", int(config.scheduleTimeout.Seconds()))
		taskOpts.ScheduleTimeout = &s
	}

	req := &v1.CreateWorkflowVersionRequest{
		Name:  workflowName,
		Tasks: []*v1.CreateTaskOpts{taskOpts},
	}

	// Build the low-level batch fn that the worker calls.
	batchFn := func(items []pkgWorker.BatchActionItem) ([]interface{}, error) {
		// Sort items are already ordered by index from the worker.
		// Parse each item's input from its context.
		inputSlice := reflect.MakeSlice(itemSliceType, len(items), len(items))
		for i, item := range items {
			inputVal := reflect.New(inputElemType)
			if err := item.Ctx.WorkflowInput(inputVal.Interface()); err != nil {
				return nil, fmt.Errorf("batch item %d: failed to parse input: %w", i, err)
			}
			inputSlice.Index(i).Set(inputVal.Elem())
		}

		// Call the user fn.
		var callArgs []reflect.Value
		if hasCtxParam {
			// Use the first item's context as the batch context.
			var batchCtx Context
			if len(items) > 0 {
				batchCtx = items[0].Ctx
			}
			callArgs = []reflect.Value{reflect.ValueOf(batchCtx), inputSlice}
		} else {
			callArgs = []reflect.Value{inputSlice}
		}

		results := fnValue.Call(callArgs)

		if !results[1].IsNil() {
			return nil, results[1].Interface().(error)
		}

		outputSlice := results[0]
		outputs := make([]interface{}, outputSlice.Len())
		for i := 0; i < outputSlice.Len(); i++ {
			outputs[i] = outputSlice.Index(i).Interface()
		}
		return outputs, nil
	}

	return &StandaloneBatchTask{
		name:     workflowName,
		taskName: workflowName,
		v0Client: c.legacyClient,
		req:      req,
		batchFn:  batchFn,
	}
}
