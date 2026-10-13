//go:build !go1.27

package hatchet

import (
	"reflect"
	"slices"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func validateTaskFn(fnType reflect.Type, what string) {
	if fnType.Kind() != reflect.Func {
		panic(what + " must be a function")
	}
	if fnType.NumIn() != 2 {
		panic(what + " must have exactly 2 parameters: (ctx hatchet.Context, input T)")
	}
	if fnType.NumOut() != 2 {
		panic(what + " must return exactly 2 values: (output T, err error)")
	}
	if !fnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic(what + ": second return value must be error")
	}
}

func callReflected(fnValue reflect.Value, ctx Context, input any) (any, error) {
	fnType := fnValue.Type()
	contextArg := reflect.ValueOf(ctx)
	durableContextType := reflect.TypeOf((*worker.DurableHatchetContext)(nil)).Elem()
	if fnType.In(0).Implements(durableContextType) || fnType.In(0) == durableContextType {
		contextArg = reflect.ValueOf(worker.NewDurableHatchetContext(ctx))
	}

	results := fnValue.Call([]reflect.Value{contextArg, convertInputToType(input, fnType.In(1))})

	var err error
	if !results[1].IsNil() {
		err = results[1].Interface().(error)
	}
	return results[0].Interface(), err
}

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
	validateTaskFn(fnType, "fn")

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

	wrapper := func(ctx Context, input any) (any, error) {
		return callReflected(fnValue, ctx, input)
	}

	if config.isDurable {
		durableWrapper := func(ctx worker.DurableHatchetContext, input any) (any, error) {
			return wrapper(ctx, input)
		}
		durableDecl := w.declaration.DurableTask(config.createOpts(name), durableWrapper)
		durableDecl.EvictionPolicy = config.evictionOpts()
	} else {
		w.declaration.Task(config.createOpts(name), wrapper)
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
	return w.NewTask(name, fn, append(slices.Clone(options), withDurable())...)
}

// OnFailure sets a failure handler for the workflow.
// The handler will be called when any task in the workflow fails.
//
// Function signatures are validated at runtime using reflection.
func (w *Workflow) OnFailure(fn any) {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()
	validateTaskFn(fnType, "onFailure function")

	contextType := reflect.TypeOf((*Context)(nil)).Elem()
	if !fnType.In(0).Implements(contextType) && fnType.In(0) != contextType {
		panic("first parameter must be Context")
	}

	w.declaration.OnFailure(
		create.WorkflowOnFailureTask[any, any]{},
		func(ctx Context, input any) (any, error) {
			return callReflected(fnValue, ctx, input)
		},
	)
}
