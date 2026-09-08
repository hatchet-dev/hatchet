// Define workflows that can declare tasks and be run, scheduled, and so on.
// Transform functions into Hatchet tasks using a clean, reflection-based API.
//
// # Basic Usage
//
//	client, err := hatchet.NewClient()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	workflow := client.NewWorkflow("my-workflow",
//		hatchet.WithWorkflowConcurrency(hatchet.Concurrency{
//			Expression: "input.userId",
//			MaxRuns:    5,
//		}))
//	fmt.Printf("Workflow name: %s\n", workflow.Name()) // Includes namespace if set
//
//	task1 := workflow.NewTask("task-1", MyTaskFunction)
//	task2 := workflow.NewTask("task-2", MyOtherTaskFunction,
//		hatchet.WithParents(task1))
//
//	worker, err := client.NewWorker("worker-name", hatchet.WithWorkflows(workflow))
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	err = worker.StartBlocking(ctx)
//
// # Examples
//
// For comprehensive examples demonstrating various Hatchet features, see:
//
//   - Basic workflow with a single task: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/simple
//   - Complex workflows with task dependencies: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/dag
//   - Conditional task execution and branching: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/conditions
//   - Triggered by external events: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/events
//   - Time-based workflow scheduling: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/cron
//   - Error handling and parallel execution: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/retries-concurrency
//   - Control execution rate per resource: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/rate-limiting
//   - Process multiple items efficiently: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/bulk-operations
//   - Nested workflow execution: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/child-workflows
//   - Worker affinity and state management: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/sticky-workers
//   - Long-running tasks with state persistence: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/durable
//   - Real-time data processing: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/streaming
//   - Task execution prioritization: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/priority
//   - Task timeout handling: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/timeouts
//   - Workflow and task cancellation: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/cancellations
//   - Error recovery and cleanup: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples/on-failure
//
// View all examples: https://github.com/hatchet-dev/hatchet/tree/main/sdks/go/examples
//
// # Documentation for agents
//
// The linked pages serve plain markdown for tools and agents. Full docs index: https://docs.hatchet.run/llms.txt
//
// Setup and local development:
//
//   - Quickstart: https://docs.hatchet.run/v1/quickstart.md
//   - Running Hatchet locally: https://docs.hatchet.run/v1/running-locally.md
//   - Embedded mode: https://docs.hatchet.run/v1/embedded.md
//
// Core concepts:
//
//   - Tasks: https://docs.hatchet.run/v1/tasks.md
//   - Workers: https://docs.hatchet.run/v1/workers.md
//   - Running tasks: https://docs.hatchet.run/v1/running-your-task.md
//   - DAGs: https://docs.hatchet.run/v1/directed-acyclic-graphs.md
//   - Durable execution: https://docs.hatchet.run/v1/durable-execution.md
//
// Flow control:
//
//   - Concurrency: https://docs.hatchet.run/v1/concurrency.md
//   - Rate limits: https://docs.hatchet.run/v1/rate-limits.md
//   - Retries: https://docs.hatchet.run/v1/retry-policies.md
//   - Idempotency: https://docs.hatchet.run/v1/idempotency.md
//   - CEL expressions: https://docs.hatchet.run/v1/cel-expressions.md
//
// Go SDK reference (overview: https://docs.hatchet.run/reference/go.md):
//
//   - CEL: https://docs.hatchet.run/reference/go/feature-clients/cel.md (guide: https://docs.hatchet.run/v1/cel-expressions.md)
//   - Crons: https://docs.hatchet.run/reference/go/feature-clients/crons.md (guide: https://docs.hatchet.run/v1/cron-runs.md)
//   - Filters: https://docs.hatchet.run/reference/go/feature-clients/filters.md (guide: https://docs.hatchet.run/v1/events.md)
//   - Logs: https://docs.hatchet.run/reference/go/feature-clients/logs.md (guide: https://docs.hatchet.run/v1/logging.md)
//   - Metrics: https://docs.hatchet.run/reference/go/feature-clients/metrics.md (guide: https://docs.hatchet.run/v1/prometheus-metrics.md)
//   - Rate Limits: https://docs.hatchet.run/reference/go/feature-clients/ratelimits.md (guide: https://docs.hatchet.run/v1/rate-limits.md)
//   - Runs: https://docs.hatchet.run/reference/go/feature-clients/runs.md (guide: https://docs.hatchet.run/v1/running-your-task.md)
//   - Scheduled Runs: https://docs.hatchet.run/reference/go/feature-clients/schedules.md (guide: https://docs.hatchet.run/v1/scheduled-runs.md)
//   - Webhooks: https://docs.hatchet.run/reference/go/feature-clients/webhooks.md (guide: https://docs.hatchet.run/v1/webhooks.md)
//   - Workers: https://docs.hatchet.run/reference/go/feature-clients/workers.md (guide: https://docs.hatchet.run/v1/workers.md)
//   - Workflows: https://docs.hatchet.run/reference/go/feature-clients/workflows.md (guide: https://docs.hatchet.run/v1/tasks.md)
//
//hatchet:agent-docs-generated (do not edit; regenerate with `go run ./docs/generator`)
package hatchet

import (
	"time"

	pkgWorker "github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
)

// Context represents the execution context passed to task functions.
// It provides access to workflow metadata, retry information, and other execution details.
type Context = pkgWorker.HatchetContext

// DurableContext represents the execution context for durable tasks.
// It extends Context with additional methods for durable operations like SleepFor.
type DurableContext = pkgWorker.DurableHatchetContext

// WaitResult holds the results of a DurableContext.WaitFor call, keyed by condition.
type WaitResult = pkgWorker.WaitResult

// SingleWaitResult holds the result of a single-condition durable wait such as
// DurableContext.SleepFor or DurableContext.WaitForEvent.
type SingleWaitResult = pkgWorker.SingleWaitResult

// Condition helpers for workflow task conditions

// Condition is a condition used with WithWaitFor and WithSkipIf to gate task execution.
// Build conditions with SleepCondition, UserEventCondition, ParentCondition, OrCondition,
// and AndCondition.
type Condition = condition.Condition

// UserEventConditionOpt configures a UserEventCondition.
type UserEventConditionOpt = condition.UserEventConditionOpt

// SleepCondition creates a condition that waits for a specified duration.
func SleepCondition(duration time.Duration) Condition {
	return condition.SleepCondition(duration)
}

// UserEventCondition creates a condition that waits for a user event.
func UserEventCondition(eventKey, expression string, opts ...UserEventConditionOpt) Condition {
	return condition.UserEventCondition(eventKey, expression, opts...)
}

// WithEventScope restricts a user event condition to events pushed with a matching scope.
func WithEventScope(scope string) UserEventConditionOpt {
	return condition.WithEventScope(scope)
}

// WithConsiderEventsSince makes a user event condition also match events pushed
// after the given time but before the wait was registered (event lookback).
// Requires WithEventScope to be set as well.
func WithConsiderEventsSince(since time.Time) UserEventConditionOpt {
	return condition.WithConsiderEventsSince(since)
}

// ParentCondition creates a condition based on a parent task's output.
func ParentCondition(task *Task, expression string) Condition {
	return condition.ParentCondition(task, expression)
}

// OrCondition creates a condition that is satisfied when any of the provided conditions are met.
func OrCondition(conditions ...Condition) Condition {
	return condition.Or(conditions...)
}

// AndCondition creates a condition that is satisfied when all of the provided conditions are met.
func AndCondition(conditions ...Condition) Condition {
	return condition.Conditions(conditions...)
}

// EventUnmarshaller is implemented by the result of DurableContext.WaitForEvent.
// Use EventInto to extract the event payload.
type EventUnmarshaller interface {
	Unmarshal(dest any) error
}

// EventInto extracts the event payload from a WaitForEvent result into dest.
//
//	event, err := ctx.WaitForEvent("approval:decision", "")
//	if err != nil { return err }
//	var data map[string]interface{}
//	if err := hatchet.EventInto(event, &data); err != nil { return err }
func EventInto(event EventUnmarshaller, dest any) error {
	return event.Unmarshal(dest)
}
