package task

import (
	"fmt"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// TaskFn is the function that will be executed when the task runs.
// It takes an input and a Hatchet context and returns an output and an error.
type TaskFn[I any, O any] func(input I, ctx worker.HatchetContext) (*O, error)

// CreateOpts is the options for creating a task.
type CreateOpts[I any, O any] struct {
	// Name is the unique identifier for the task
	Name string

	// ExecutionTimeout specifies the maximum duration a task can run before being terminated
	ExecutionTimeout string

	// ScheduleTimeout specifies the maximum time a task can wait to be scheduled
	ScheduleTimeout string

	// Retries defines the number of times to retry a failed task
	Retries int32

	// RetryBackoffFactor is the multiplier for increasing backoff between retries
	RetryBackoffFactor *float32

	// RetryMaxBackoffSeconds is the maximum backoff duration in seconds between retries
	RetryMaxBackoffSeconds *int32

	// RateLimits define constraints on how frequently the task can be executed
	RateLimits []*types.RateLimit

	// WorkerLabels specify requirements for workers that can execute this task
	WorkerLabels map[string]*types.DesiredWorkerLabel

	// BackoffFactor is the multiplier for increasing delay between task execution attempts
	BackoffFactor *float32

	// BackoffMaxSeconds is the maximum delay in seconds between task execution attempts
	BackoffMaxSeconds *int32

	// Concurrency defines constraints on how many instances of this task can run simultaneously
	Concurrency []*types.Concurrency

	// Conditions specifies when this task should be executed
	Conditions *types.TaskConditions

	// Parents defines the tasks that must complete before this task can start
	Parents []*TaskDeclaration[I, O]

	// Fn is the function to execute when the task runs
	Fn TaskFn[I, O]
}

type TaskBase interface {
	Dump() *contracts.CreateTaskOpts
}

type TaskShared[I any, O any] struct {
	// ExecutionTimeout specifies the maximum duration a task can run before being terminated
	ExecutionTimeout string

	// ScheduleTimeout specifies the maximum time a task can wait to be scheduled
	ScheduleTimeout string

	// Retries defines the number of times to retry a failed task
	Retries int32

	// RetryBackoffFactor is the multiplier for increasing backoff between retries
	RetryBackoffFactor *float32

	// RetryMaxBackoffSeconds is the maximum backoff duration in seconds between retries
	RetryMaxBackoffSeconds *int32

	// RateLimits define constraints on how frequently the task can be executed
	RateLimits []*types.RateLimit

	// WorkerLabels specify requirements for workers that can execute this task
	WorkerLabels map[string]*types.DesiredWorkerLabel

	// BackoffFactor is the multiplier for increasing delay between task execution attempts
	BackoffFactor *float32

	// BackoffMaxSeconds is the maximum delay in seconds between task execution attempts
	BackoffMaxSeconds *int32

	// Concurrency defines constraints on how many instances of this task can run simultaneously
	Concurrency []*types.Concurrency

	// Fn is the function to execute when the task runs
	Fn func(input I, ctx worker.HatchetContext) (*O, error)
}

// TaskDeclaration represents a task configuration that can be added to a workflow.
type TaskDeclaration[I any, O any] struct {
	TaskBase
	TaskShared[I, O]

	// The friendly name of the task
	Name string

	// The tasks that must successfully complete before this task can start
	Parents []*TaskDeclaration[I, O]

	// Conditions specifies when this task should be executed
	Conditions *types.TaskConditions

	// The function to execute when the task runs
	Fn func(input I, ctx worker.HatchetContext) (*O, error)

	// Concurrency defines constraints on how many instances of this task can run simultaneously
	// and group key expression to evaluate when determining if a task can run
	Concurrency []*types.Concurrency
}

// OnFailureTaskDeclaration represents a task that will be executed if
// any tasks in the workflow fail.
type OnFailureTaskDeclaration[I any, O any] struct {
	TaskBase
	TaskShared[I, O]

	// The function to execute when any tasks in the workflow have failed
	Fn func(input I, ctx worker.HatchetContext) (*O, error)
}

// NewTaskDeclaration creates a new task declaration with the specified options.
func NewTaskDeclaration[I any, O any](opts CreateOpts[I, O]) *TaskDeclaration[I, O] {
	return &TaskDeclaration[I, O]{
		Name:       opts.Name,
		Fn:         opts.Fn,
		Parents:    opts.Parents,
		Conditions: opts.Conditions,

		TaskShared: TaskShared[I, O]{
			ExecutionTimeout:       opts.ExecutionTimeout,
			ScheduleTimeout:        opts.ScheduleTimeout,
			Retries:                opts.Retries,
			RetryBackoffFactor:     opts.RetryBackoffFactor,
			RetryMaxBackoffSeconds: opts.RetryMaxBackoffSeconds,
			RateLimits:             opts.RateLimits,
			WorkerLabels:           opts.WorkerLabels,
			BackoffFactor:          opts.BackoffFactor,
			BackoffMaxSeconds:      opts.BackoffMaxSeconds,
			Concurrency:            opts.Concurrency,
		},
	}
}

func makeContractTaskOpts[I any, O any](t *TaskShared[I, O]) *contracts.CreateTaskOpts {

	rateLimits := make([]*contracts.CreateTaskRateLimit, len(t.RateLimits))
	for j, rateLimit := range t.RateLimits {
		rateLimits[j] = &contracts.CreateTaskRateLimit{
			Key:     rateLimit.Key,
			KeyExpr: rateLimit.KeyExpr,
		}
	}

	concurrencyOpts := make([]*contracts.Concurrency, len(t.Concurrency))
	for j, concurrency := range t.Concurrency {
		concurrencyOpts[j] = &contracts.Concurrency{
			Expression: concurrency.Expression,
			MaxRuns:    concurrency.MaxRuns,
		}

		if concurrency.LimitStrategy != nil {
			strategy := *concurrency.LimitStrategy
			strategyInt := contracts.ConcurrencyLimitStrategy_value[string(strategy)]
			strategyEnum := contracts.ConcurrencyLimitStrategy(strategyInt)
			concurrencyOpts[j].LimitStrategy = &strategyEnum
		}
	}

	taskOpts := &contracts.CreateTaskOpts{
		Timeout:         t.ExecutionTimeout,
		ScheduleTimeout: &t.ScheduleTimeout,
		Retries:         t.Retries,
		RateLimits:      rateLimits,
		// TODO WorkerLabels:      task.WorkerLabels,
		BackoffFactor:     t.BackoffFactor,
		BackoffMaxSeconds: t.BackoffMaxSeconds,
		Concurrency:       concurrencyOpts,
	}

	return taskOpts
}

// Dump converts the task declaration into a protobuf request.
func (t *TaskDeclaration[I, O]) Dump(workflowName string) *contracts.CreateTaskOpts {

	base := makeContractTaskOpts(&t.TaskShared)

	base.ReadableId = t.Name
	base.Action = fmt.Sprintf("%s:%s", workflowName, t.Name)

	base.Parents = make([]string, len(t.Parents))
	for i, parent := range t.Parents {
		base.Parents[i] = parent.Name
	}

	// TODO: Conditions

	return base
}

// Dump converts the on failure task declaration into a protobuf request.
func (t *OnFailureTaskDeclaration[I, O]) Dump(workflowName string) *contracts.CreateTaskOpts {
	base := makeContractTaskOpts(&t.TaskShared)

	base.ReadableId = "on-failure"
	base.Action = fmt.Sprintf("%s:%s", workflowName, "on-failure")

	return base
}
