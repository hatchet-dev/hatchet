// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package create

import (
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

// TaskDefaults defines default configuration values for tasks within a workflow.
type TaskDefaults struct {
	// (optional) ExecutionTimeout specifies the maximum duration a task can run after starting before being terminated
	ExecutionTimeout time.Duration

	// (optional) ScheduleTimeout specifies the maximum time a task can wait in the queue to be scheduled
	ScheduleTimeout time.Duration

	// (optional) Retries defines the number of times to retry a failed task
	Retries int32

	// (optional) RetryBackoffFactor is the multiplier for increasing backoff between retries
	RetryBackoffFactor float32

	// (optional) RetryMaxBackoffSeconds is the maximum backoff duration in seconds between retries
	RetryMaxBackoffSeconds int32
}

// WorkflowCreateOpts contains configuration options for creating a new workflow.
type WorkflowCreateOpts[I any] struct {
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

	// (optional) The JSON-serialized input for cron workflows (defaults to "{}" if nil)
	CronInput *string

	// (optional) Concurrency settings to control parallel execution
	Concurrency []types.Concurrency

	// (optional) Strategy for sticky execution of workflow runs
	StickyStrategy *types.StickyStrategy

	// (optional) Default settings for all tasks within this workflow
	TaskDefaults *TaskDefaults

	// (optional) The key to use for the output of the workflow (i.e. the name of the fn where O is the output type)
	OutputKey *string

	// (optional) The default priority for tasks in this workflow
	DefaultPriority *int32

	DefaultFilters []types.DefaultFilter
}
