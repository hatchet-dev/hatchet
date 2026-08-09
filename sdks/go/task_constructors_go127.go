//go:build go1.27

// This file holds the generic, reflection-free task constructors that require Go 1.27
// generic methods. When built with Go 1.26 or earlier, task_constructors_pre_go127.go is
// compiled instead, providing the same method names with the older reflection-based
// (fn any) signatures. See task_constructors_pre_go127.go for the fallback.

package hatchet

import (
	"encoding/json"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/sdks/go/internal"
)

// decodeTaskInput converts the raw workflow input (typically a map[string]any decoded
// from JSON) into the task's declared input type I, without reflection. A direct type
// assertion covers the common case; otherwise a JSON round-trip fills a fresh I.
func decodeTaskInput[I any](input any) (I, error) {
	var out I
	if input == nil {
		return out, nil
	}
	if v, ok := input.(I); ok {
		return v, nil
	}
	b, err := json.Marshal(input)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, err
	}
	return out, nil
}

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

	wrapper := func(ctx Context, input any) (any, error) {
		in, err := decodeTaskInput[I](input)
		if err != nil {
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

	durableWrapper := func(ctx worker.DurableHatchetContext, input any) (any, error) {
		in, err := decodeTaskInput[I](input)
		if err != nil {
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
