//go:build go1.27

// Generic, reflection-free task constructors, using Go 1.27 generic methods.

package hatchet

import (
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/sdks/go/internal"
)

// NewTask transforms a typed function into a Hatchet task that runs as part of a workflow.
//
// The input and output types are captured as type parameters, so the function signature is
// checked at compile time and invoked directly — no reflection. Type parameters are inferred
// from the function literal, so existing call sites need no changes.
//
//	func(ctx hatchet.Context, input In) (Out, error)
func (w *Workflow) NewTask[I, O any](name string, fn func(ctx Context, input I) (O, error), options ...TaskOption) *Task {
	if name == "" {
		panic("task name cannot be empty")
	}

	if fn == nil {
		panic("task '" + name + "' has a nil input function")
	}

	config := &taskConfig{}

	for _, opt := range options {
		opt(config)
	}

	wrapper := func(ctx Context, _ any) (any, error) {
		var in I
		if err := ctx.WorkflowInput(&in); err != nil {
			return nil, err
		}
		return fn(ctx, in)
	}

	taskOpts := create.WorkflowTask[any, any]{
		Name:                   name,
		Retries:                config.retries,
		RetryBackoffFactor:     config.retryBackoffFactor,
		RetryMaxBackoffSeconds: config.retryMaxBackoffSeconds,
		ExecutionTimeout:       config.executionTimeout,
		ScheduleTimeout:        config.scheduleTimeout,
		Concurrency:            config.concurrency,
		RateLimits:             config.rateLimits,
		Parents:                config.parents,
		WaitFor:                config.waitFor,
		SkipIf:                 config.skipIf,
		SlotCost:               config.slotCost,
	}

	w.declaration.Task(taskOpts, wrapper)

	return &Task{name: name}
}

// NewDurableTask transforms a typed function into a durable Hatchet task that runs as part
// of a workflow. Like NewTask, the input and output types are type parameters checked at
// compile time and invoked without reflection.
//
//	func(ctx hatchet.DurableContext, input In) (Out, error)
func (w *Workflow) NewDurableTask[I, O any](name string, fn func(ctx DurableContext, input I) (O, error), options ...TaskOption) *Task {
	if name == "" {
		panic("task name cannot be empty")
	}

	if fn == nil {
		panic("task '" + name + "' has a nil input function")
	}

	config := &taskConfig{}

	for _, opt := range options {
		opt(config)
	}

	durableWrapper := func(ctx worker.DurableHatchetContext, _ any) (any, error) {
		var in I
		if err := ctx.WorkflowInput(&in); err != nil {
			return nil, err
		}
		return fn(ctx, in)
	}

	taskOpts := create.WorkflowTask[any, any]{
		Name:                   name,
		Retries:                config.retries,
		RetryBackoffFactor:     config.retryBackoffFactor,
		RetryMaxBackoffSeconds: config.retryMaxBackoffSeconds,
		ExecutionTimeout:       config.executionTimeout,
		ScheduleTimeout:        config.scheduleTimeout,
		Concurrency:            config.concurrency,
		RateLimits:             config.rateLimits,
		Parents:                config.parents,
		WaitFor:                config.waitFor,
		SkipIf:                 config.skipIf,
		SlotCost:               config.slotCost,
	}

	durableDecl := w.declaration.DurableTask(taskOpts, durableWrapper)
	if config.evictionPolicy != nil {
		durableDecl.EvictionPolicy = &internal.EvictionPolicyOpts{
			TTL:                   config.evictionPolicy.TTL,
			AllowCapacityEviction: config.evictionPolicy.AllowCapacityEviction,
			Priority:              config.evictionPolicy.Priority,
		}
	}

	return &Task{name: name}
}

// OnFailure sets a typed failure handler for the workflow. It runs when any task in the
// workflow fails and receives the same input the workflow was triggered with. The input
// and output types are type parameters, inferred from the handler and invoked without
// reflection.
//
//	func(ctx hatchet.Context, input In) (Out, error)
func (w *Workflow) OnFailure[I, O any](fn func(ctx Context, input I) (O, error)) {
	wrapper := func(ctx Context, _ any) (any, error) {
		var in I
		if err := ctx.WorkflowInput(&in); err != nil {
			return nil, err
		}
		return fn(ctx, in)
	}

	w.declaration.OnFailure(create.WorkflowOnFailureTask[any, any]{}, wrapper)
}
