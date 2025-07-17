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
	pkgWorker "github.com/hatchet-dev/hatchet/pkg/worker"
)

// Context represents the execution context passed to task functions.
// It provides access to workflow metadata, retry information, and other execution details.
type Context = pkgWorker.HatchetContext

// DurableContext represents the execution context for durable tasks.
// It extends Context with additional methods for durable operations like SleepFor.
type DurableContext = pkgWorker.DurableHatchetContext