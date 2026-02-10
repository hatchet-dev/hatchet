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
//		hatchet.WithWorkflowConcurrency(types.Concurrency{
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
package hatchet

import (
	"time"

	pkgWorker "github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
	"github.com/hatchet-dev/hatchet/pkg/worker/eviction"
)

// Context represents the execution context passed to task functions.
// It provides access to workflow metadata, retry information, and other execution details.
type Context = pkgWorker.HatchetContext

// DurableContext represents the execution context for durable tasks.
// It extends Context with additional methods for durable operations like SleepFor.
type DurableContext = pkgWorker.DurableHatchetContext

// Condition helpers for workflow task conditions

// SleepCondition creates a condition that waits for a specified duration.
func SleepCondition(duration time.Duration) condition.Condition {
	return condition.SleepCondition(duration)
}

// UserEventCondition creates a condition that waits for a user event.
func UserEventCondition(eventKey, expression string) condition.Condition {
	return condition.UserEventCondition(eventKey, expression)
}

// ParentCondition creates a condition based on a parent task's output.
func ParentCondition(task *Task, expression string) condition.Condition {
	return condition.ParentCondition(task, expression)
}

// OrCondition creates a condition that is satisfied when any of the provided conditions are met.
func OrCondition(conditions ...condition.Condition) condition.Condition {
	return condition.Or(conditions...)
}

// AndCondition creates a condition that is satisfied when all of the provided conditions are met.
func AndCondition(conditions ...condition.Condition) condition.Condition {
	return condition.Conditions(conditions...)
}

// Eviction types re-exported for convenience.

// EvictionPolicy defines task-scoped eviction parameters for durable tasks.
type EvictionPolicy = eviction.Policy

// EvictionManagerConfig holds per-worker eviction manager settings.
type EvictionManagerConfig = eviction.ManagerConfig

// ErrEvicted is the error cause set on a durable run's context when it is evicted.
var ErrEvicted = eviction.ErrEvicted

// DefaultEvictionPolicy returns sensible defaults for durable task eviction.
func DefaultEvictionPolicy() *EvictionPolicy {
	return eviction.DefaultPolicy()
}

// DefaultEvictionManagerConfig returns sensible defaults for the eviction manager.
func DefaultEvictionManagerConfig() EvictionManagerConfig {
	return eviction.DefaultManagerConfig()
}
