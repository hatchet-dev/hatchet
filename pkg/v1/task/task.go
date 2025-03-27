package task

import (
	"fmt"
	"time"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

type NamedTaskImpl struct {
	Name string
}

type TaskBase interface {
	Dump(workflowName string, taskDefaults *create.TaskDefaults) *contracts.CreateTaskOpts
}

type TaskShared struct {
	// ExecutionTimeout specifies the maximum duration a task can run before being terminated
	ExecutionTimeout *time.Duration

	// ScheduleTimeout specifies the maximum time a task can wait to be scheduled
	ScheduleTimeout *time.Duration

	// Retries defines the number of times to retry a failed task
	Retries *int32

	// RetryBackoffFactor is the multiplier for increasing backoff between retries
	RetryBackoffFactor *float32

	// RetryMaxBackoffSeconds is the maximum backoff duration in seconds between retries
	RetryMaxBackoffSeconds *int32

	// RateLimits define constraints on how frequently the task can be executed
	RateLimits []*types.RateLimit

	// WorkerLabels specify requirements for workers that can execute this task
	WorkerLabels map[string]*types.DesiredWorkerLabel

	// Concurrency defines constraints on how many instances of this task can run simultaneously
	Concurrency []*types.Concurrency

	// The function to execute when the task runs
	// must be a function that takes an input and a Hatchet context and returns an output and an error
	Fn interface{}
}

// TaskDeclaration represents a standard (non-durable) task configuration that can be added to a workflow.
type TaskDeclaration[I any] struct {
	TaskBase
	NamedTaskImpl
	TaskShared

	// The friendly name of the task
	Name string

	// The tasks that must successfully complete before this task can start
	Parents []string

	// Conditions specifies when this task should be executed
	Conditions *types.TaskConditions

	// The function to execute when the task runs
	// must be a function that takes an input and a Hatchet context and returns an output and an error
	Fn interface{}
}

// DurableTaskDeclaration represents a durable task configuration that can be added to a workflow.
// Durable tasks can use the DurableHatchetContext for operations that persist across worker restarts.
type DurableTaskDeclaration[I any] struct {
	TaskBase
	NamedTaskImpl
	TaskShared

	// The friendly name of the task
	Name string

	// The tasks that must successfully complete before this task can start
	Parents []string

	// Concurrency defines constraints on how many instances of this task can run simultaneously
	// and group key expression to evaluate when determining if a task can run
	Concurrency []*types.Concurrency

	// Conditions specifies when this task should be executed
	Conditions *types.TaskConditions

	// The function to execute when the task runs
	// must be a function that takes an input and a DurableHatchetContext and returns an output and an error
	Fn interface{}
}

// OnFailureTaskDeclaration represents a task that will be executed if
// any tasks in the workflow fail.
type OnFailureTaskDeclaration[I any] struct {
	TaskBase
	TaskShared

	// The function to execute when any tasks in the workflow have failed
	Fn interface{}
}

func makeContractTaskOpts(t *TaskShared, taskDefaults *create.TaskDefaults) *contracts.CreateTaskOpts {
	taskOpts := &contracts.CreateTaskOpts{
		RateLimits:  make([]*contracts.CreateTaskRateLimit, len(t.RateLimits)),
		Concurrency: make([]*contracts.Concurrency, len(t.Concurrency)),
	}

	// Only set Retries if it's not nil
	if t.Retries != nil {
		taskOpts.Retries = *t.Retries
	}

	for j, rateLimit := range t.RateLimits {
		taskOpts.RateLimits[j] = &contracts.CreateTaskRateLimit{
			Key:     rateLimit.Key,
			KeyExpr: rateLimit.KeyExpr,
		}
	}

	for j, concurrency := range t.Concurrency {
		concurrencyOpts := &contracts.Concurrency{
			Expression: concurrency.Expression,
			MaxRuns:    concurrency.MaxRuns,
		}

		if concurrency.LimitStrategy != nil {
			strategy := *concurrency.LimitStrategy
			strategyInt := contracts.ConcurrencyLimitStrategy_value[string(strategy)]
			strategyEnum := contracts.ConcurrencyLimitStrategy(strategyInt)
			concurrencyOpts.LimitStrategy = &strategyEnum
		}

		taskOpts.Concurrency[j] = concurrencyOpts
	}

	if t.ExecutionTimeout != nil {
		executionTimeout := t.ExecutionTimeout.String()
		taskOpts.Timeout = executionTimeout
	}

	if t.ScheduleTimeout != nil {
		scheduleTimeout := t.ScheduleTimeout.String()
		taskOpts.ScheduleTimeout = &scheduleTimeout
	}

	// Apply workflow task defaults if they are not set
	if taskDefaults != nil {
		if t.Retries == nil && taskDefaults.Retries != 0 {
			taskOpts.Retries = taskDefaults.Retries
		}

		if t.ExecutionTimeout == nil && taskDefaults.ExecutionTimeout != 0 {
			executionTimeout := taskDefaults.ExecutionTimeout.String()
			taskOpts.Timeout = executionTimeout
		}

		if t.ScheduleTimeout == nil && taskDefaults.ScheduleTimeout != 0 {
			scheduleTimeout := taskDefaults.ScheduleTimeout.String()
			taskOpts.ScheduleTimeout = &scheduleTimeout
		}

		if t.RetryBackoffFactor == nil && taskDefaults.RetryBackoffFactor != 0 {
			taskOpts.BackoffFactor = &taskDefaults.RetryBackoffFactor
		}

		if t.RetryMaxBackoffSeconds == nil && taskDefaults.RetryMaxBackoffSeconds != 0 {
			taskOpts.BackoffMaxSeconds = &taskDefaults.RetryMaxBackoffSeconds
		}
	}

	return taskOpts
}

// Dump converts the task declaration into a protobuf request.
func (t *TaskDeclaration[I]) Dump(workflowName string, taskDefaults *create.TaskDefaults) *contracts.CreateTaskOpts {
	base := makeContractTaskOpts(&t.TaskShared, taskDefaults)
	base.ReadableId = t.Name
	base.Action = fmt.Sprintf("%s:%s", workflowName, t.Name)
	base.Parents = make([]string, len(t.Parents))
	copy(base.Parents, t.Parents)
	return base
}

func (t *DurableTaskDeclaration[I]) Dump(workflowName string, taskDefaults *create.TaskDefaults) *contracts.CreateTaskOpts {
	base := makeContractTaskOpts(&t.TaskShared, taskDefaults)
	base.ReadableId = t.Name
	base.Action = fmt.Sprintf("%s:%s", workflowName, t.Name)
	base.Parents = make([]string, len(t.Parents))
	copy(base.Parents, t.Parents)
	return base
}

// Dump converts the on failure task declaration into a protobuf request.
func (t *OnFailureTaskDeclaration[I]) Dump(workflowName string, taskDefaults *create.TaskDefaults) *contracts.CreateTaskOpts {
	base := makeContractTaskOpts(&t.TaskShared, taskDefaults)

	base.ReadableId = "on-failure"
	base.Action = fmt.Sprintf("%s:%s", workflowName, "on-failure")

	return base
}

// Implement GetName for TaskDeclaration
func (t *TaskDeclaration[I]) GetName() string {
	return t.Name
}

// Implement GetName for DurableTaskDeclaration
func (t *DurableTaskDeclaration[I]) GetName() string {
	return t.Name
}

// Implement GetName for NamedTask
func (t *NamedTaskImpl) GetName() string {
	return t.Name
}
