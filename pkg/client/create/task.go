package create

import (
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
)

type BaseTaskCreateOpts[I any] struct {
}

// NamedTask defines an interface for task types that have a name
type NamedTask interface {
	// GetName returns the name of the task
	GetName() string
}

type WorkflowTask[I, O any] struct {
	// (required) The name of the task and workflow
	Name string

	// (optional) The version of the workflow
	Version string

	// (optional) The human-readable description of the workflow
	Description string

	// (optional) ExecutionTimeout specifies the maximum duration a task can run before being terminated
	ExecutionTimeout time.Duration

	// (optional) ScheduleTimeout specifies the maximum time a task can wait to be scheduled
	ScheduleTimeout time.Duration

	// (optional) Retries defines the number of times to retry a failed task
	Retries int32

	// (optional) RetryBackoffFactor is the multiplier for increasing backoff between retries
	RetryBackoffFactor float32

	// (optional) RetryMaxBackoffSeconds is the maximum backoff duration in seconds between retries
	RetryMaxBackoffSeconds int32

	// (optional) RateLimits define constraints on how frequently the task can be executed
	RateLimits []*types.RateLimit

	// (optional) WorkerLabels specify requirements for workers that can execute this task
	WorkerLabels map[string]*types.DesiredWorkerLabel

	// (optional) Concurrency defines constraints on how many instances of this task can run simultaneously
	Concurrency []*types.Concurrency

	// WaitFor represents a set of conditions which must be satisfied before the task can run.
	WaitFor condition.Condition

	// SkipIf represents a set of conditions which, if satisfied, will cause the task to be skipped.
	SkipIf condition.Condition

	// CancelIf represents a set of conditions which, if satisfied, will cause the task to be canceled.
	CancelIf condition.Condition

	// (optional) Parents are the tasks that must successfully complete before this task can start
	Parents []NamedTask
}

type WorkflowOnFailureTask[I, O any] struct {
	// (optional) The version of the workflow
	Version string

	// (optional) The human-readable description of the workflow
	Description string

	// (optional) ExecutionTimeout specifies the maximum duration a task can run before being terminated
	ExecutionTimeout time.Duration

	// (optional) ScheduleTimeout specifies the maximum time a task can wait to be scheduled
	ScheduleTimeout time.Duration

	// (optional) Retries defines the number of times to retry a failed task
	Retries int32

	// (optional) RetryBackoffFactor is the multiplier for increasing backoff between retries
	RetryBackoffFactor float32

	// (optional) RetryMaxBackoffSeconds is the maximum backoff duration in seconds between retries
	RetryMaxBackoffSeconds int32

	// (optional) RateLimits define constraints on how frequently the task can be executed
	RateLimits []*types.RateLimit

	// (optional) WorkerLabels specify requirements for workers that can execute this task
	WorkerLabels map[string]*types.DesiredWorkerLabel

	// (optional) Concurrency defines constraints on how many instances of this task can run simultaneously
	Concurrency []*types.Concurrency
}

// TaskCreateOpts defines options for creating a standalone task.
// This combines both workflow and task properties in a single type.
type StandaloneTask struct {

	// (required) The name of the task and workflow
	Name string

	// (optional) The version of the workflow
	Version string

	// (optional) The human-readable description of the workflow
	Description string

	// (optional) ExecutionTimeout specifies the maximum duration a task can run before being terminated
	ExecutionTimeout time.Duration

	// (optional) ScheduleTimeout specifies the maximum time a task can wait to be scheduled
	ScheduleTimeout time.Duration

	// (optional) Retries defines the number of times to retry a failed task
	Retries int32

	// (optional) RetryBackoffFactor is the multiplier for increasing backoff between retries
	RetryBackoffFactor float32

	// (optional) RetryMaxBackoffSeconds is the maximum backoff duration in seconds between retries
	RetryMaxBackoffSeconds int32

	// (optional) RateLimits define constraints on how frequently the task can be executed
	RateLimits []*types.RateLimit

	// (optional) WorkerLabels specify requirements for workers that can execute this task
	WorkerLabels map[string]*types.DesiredWorkerLabel

	// (optional) Concurrency defines constraints on how many instances of this task can run simultaneously
	Concurrency []*types.Concurrency

	// (optional) The event names that trigger the workflow
	OnEvents []string

	// (optional) The cron expressions for scheduled workflow runs
	OnCron []string
}

// DurableTaskCreateOpts defines options for creating a standalone durable task.
// This combines both workflow and durable task properties in a single type.
type StandaloneDurableTaskCreateOpts[I, O any] struct {
	StandaloneTask
	// (required) The function to execute when the task runs
	Fn interface{}
}
