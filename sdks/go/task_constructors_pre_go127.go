//go:build !go1.27

// This file holds the reflection-based (fn any) task constructors used on Go 1.26 and
// earlier, which do not support generic methods. On Go 1.27+, task_constructors_go127.go
// is compiled instead, providing the same method names with typed, reflection-free
// signatures. The two files are mutually exclusive via the go1.27 build tag.

package hatchet

import (
	"reflect"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/sdks/go/internal"
)

// NewTask transforms a function into a Hatchet task that runs as part of a workflow.
//
// The function parameter must have the signature:
//
//	func(ctx hatchet.Context, input any) (any, error)
//
// Function signatures are validated at runtime using reflection.
func (w *Workflow) NewTask(name string, fn any, options ...TaskOption) *Task {
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

	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		panic("fn must be a function")
	}

	if fnType.NumIn() != 2 {
		panic("fn must have exactly 2 parameters: (ctx hatchet.Context, input T)")
	}

	if fnType.NumOut() != 2 {
		panic("fn must return exactly 2 values: (output T, err error)")
	}

	contextType := reflect.TypeOf((*Context)(nil)).Elem()
	durableContextType := reflect.TypeOf((*worker.DurableHatchetContext)(nil)).Elem()

	if config.isDurable {
		if !fnType.In(0).Implements(durableContextType) && fnType.In(0) != durableContextType {
			panic("first parameter for durable task must be hatchet.DurableContext")
		}
	} else {
		if !fnType.In(0).Implements(contextType) && fnType.In(0) != contextType {
			panic("first parameter must be hatchet.Context")
		}
	}

	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !fnType.Out(1).Implements(errorType) {
		panic("second return value must be error")
	}

	wrapper := func(ctx Context, input any) (any, error) {
		convertedInput := convertInputToType(input, fnType.In(1))

		var contextArg reflect.Value
		if fnType.In(0).Implements(durableContextType) || fnType.In(0) == durableContextType {
			durableCtx := worker.NewDurableHatchetContext(ctx)
			contextArg = reflect.ValueOf(durableCtx)
		} else {
			contextArg = reflect.ValueOf(ctx)
		}

		args := []reflect.Value{
			contextArg,
			convertedInput,
		}

		results := fnValue.Call(args)

		output := results[0].Interface()
		var err error
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}

		return output, err
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

	if config.isDurable {
		durableWrapper := func(ctx worker.DurableHatchetContext, input any) (any, error) {
			return wrapper(ctx, input)
		}
		durableDecl := w.declaration.DurableTask(taskOpts, durableWrapper)
		if config.evictionPolicy != nil {
			durableDecl.EvictionPolicy = &internal.EvictionPolicyOpts{
				TTL:                   config.evictionPolicy.TTL,
				AllowCapacityEviction: config.evictionPolicy.AllowCapacityEviction,
				Priority:              config.evictionPolicy.Priority,
			}
		}
	} else {
		w.declaration.Task(taskOpts, wrapper)
	}

	return &Task{name: name}
}

// NewDurableTask transforms a function into a durable Hatchet task that runs as part of a workflow.
//
// The function parameter must have the signature:
//
//	func(ctx hatchet.DurableContext, input any) (any, error)
//
// Function signatures are validated at runtime using reflection.
func (w *Workflow) NewDurableTask(name string, fn any, options ...TaskOption) *Task {
	durableOptions := make([]TaskOption, len(options), len(options)+1)
	copy(durableOptions, options)
	durableOptions = append(durableOptions, withDurable())
	return w.NewTask(name, fn, durableOptions...)
}
