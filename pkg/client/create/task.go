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
	CancelIf               condition.Condition
	SkipIf                 condition.Condition
	WaitFor                condition.Condition
	DefaultPriority        *int32
	WorkerLabels           map[string]*types.DesiredWorkerLabel
	Name                   string
	Version                string
	Description            string
	RateLimits             []*types.RateLimit
	Concurrency            []*types.Concurrency
	Parents                []NamedTask
	ScheduleTimeout        time.Duration
	ExecutionTimeout       time.Duration
	RetryMaxBackoffSeconds int32
	RetryBackoffFactor     float32
	Retries                int32
}

type WorkflowOnFailureTask[I, O any] struct {
	WorkerLabels           map[string]*types.DesiredWorkerLabel
	Version                string
	Description            string
	RateLimits             []*types.RateLimit
	Concurrency            []*types.Concurrency
	ExecutionTimeout       time.Duration
	ScheduleTimeout        time.Duration
	Retries                int32
	RetryBackoffFactor     float32
	RetryMaxBackoffSeconds int32
}

// TaskCreateOpts defines options for creating a standalone task.
// This combines both workflow and task properties in a single type.
type StandaloneTask struct {
	DefaultPriority        *int32
	WorkerLabels           map[string]*types.DesiredWorkerLabel
	Version                string
	Description            string
	Name                   string
	Concurrency            []*types.Concurrency
	DefaultFilters         []types.DefaultFilter
	OnCron                 []string
	OnEvents               []string
	RateLimits             []*types.RateLimit
	ExecutionTimeout       time.Duration
	ScheduleTimeout        time.Duration
	RetryMaxBackoffSeconds int32
	RetryBackoffFactor     float32
	Retries                int32
}

// DurableTaskCreateOpts defines options for creating a standalone durable task.
// This combines both workflow and durable task properties in a single type.
type StandaloneDurableTaskCreateOpts[I, O any] struct {
	Fn interface{}
	StandaloneTask
}
