// Package hatchet provides a Go client for the Hatchet workflow orchestration platform.
//
// Define workflows that can declare tasks and be run, scheduled, and so on.
// Transform functions into Hatchet tasks using a clean, reflection-based API.
//
// Basic usage:
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
