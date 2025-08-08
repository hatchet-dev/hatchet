// Package hatchet provides a simplified, Go-idiomatic client for the Hatchet workflow orchestration platform.
//
// This package offers a clean, reflection-based API that eliminates the need for generics
// while maintaining type safety through runtime validation.
//
// Basic usage:
//
//	client, err := hatchet.NewClient()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	workflow := client.NewWorkflow("my-workflow")
//	workflow.NewTask("task-name", MyTaskFunction)
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
	return condition.ParentCondition(task.NamedTask, expression)
}

// OrCondition creates a condition that is satisfied when any of the provided conditions are met.
func OrCondition(conditions ...condition.Condition) condition.Condition {
	return condition.Or(conditions...)
}

// AndCondition creates a condition that is satisfied when all of the provided conditions are met.
func AndCondition(conditions ...condition.Condition) condition.Condition {
	return condition.Conditions(conditions...)
}
