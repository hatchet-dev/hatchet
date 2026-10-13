//go:build go1.27

package hatchet

import (
	"github.com/hatchet-dev/hatchet/pkg/client/create"
)

func wrapTyped[C interface{ WorkflowInput(any) error }, I, O any](fn func(C, I) (O, error)) func(C, any) (any, error) {
	return func(ctx C, _ any) (any, error) {
		var in I
		if err := ctx.WorkflowInput(&in); err != nil {
			return nil, err
		}
		return fn(ctx, in)
	}
}

// NewTask transforms a typed function into a Hatchet task that runs as part of a workflow.
//
// The input and output types are type parameters inferred from the function, so the
// signature is checked at compile time and invoked directly. On Go 1.26 and earlier,
// fn is typed any and validated at runtime instead.
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

	w.declaration.Task(config.createOpts(name), wrapTyped(fn))

	return &Task{name: name}
}

// NewDurableTask transforms a typed function into a durable Hatchet task that runs as part
// of a workflow. Like NewTask, the input and output types are type parameters checked at
// compile time. On Go 1.26 and earlier, fn is typed any and validated at runtime instead.
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

	durableDecl := w.declaration.DurableTask(config.createOpts(name), wrapTyped(fn))
	durableDecl.EvictionPolicy = config.evictionOpts()

	return &Task{name: name}
}

// OnFailure sets a typed failure handler for the workflow. It runs when any task in the
// workflow fails and receives the same input the workflow was triggered with. On Go 1.26
// and earlier, fn is typed any and validated at runtime instead.
//
//	func(ctx hatchet.Context, input In) (Out, error)
func (w *Workflow) OnFailure[I, O any](fn func(ctx Context, input I) (O, error)) {
	w.declaration.OnFailure(create.WorkflowOnFailureTask[any, any]{}, wrapTyped(fn))
}
