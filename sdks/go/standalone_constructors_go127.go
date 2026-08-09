//go:build go1.27

// Generic, reflection-free standalone task constructors (Go 1.27+). The Go 1.26 and
// earlier fallback with the same method names lives in standalone_constructors_pre_go127.go.

package hatchet

// NewStandaloneTask creates a standalone task that can be triggered independently.
// This is a specialized workflow containing only one task, making it easier to create
// simple single-task workflows without the workflow boilerplate.
//
// The function parameter must have the signature:
//
//	func(ctx hatchet.Context, input In) (Out, error)
//
// The input and output types are type parameters, checked at compile time and invoked
// without reflection. They are inferred from the function, so existing call sites are
// unchanged.
//
// Options can be any combination of WorkflowOption and TaskOption.
func (c *Client) NewStandaloneTask[I, O any](name string, fn func(ctx Context, input I) (O, error), options ...StandaloneTaskOption) *StandaloneTask {
	if name == "" {
		panic("standalone task name cannot be empty")
	}

	workflowOptions, taskOptions := splitStandaloneOptions(name, options)

	workflow := c.NewWorkflow(name, workflowOptions...)
	task := workflow.NewTask(name, fn, taskOptions...)

	return &StandaloneTask{
		workflow: workflow,
		task:     task,
	}
}

// NewStandaloneDurableTask creates a standalone durable task that can be triggered
// independently. Like NewStandaloneTask, the input and output types are type parameters
// checked at compile time and invoked without reflection.
//
//	func(ctx hatchet.DurableContext, input In) (Out, error)
//
// Options can be any combination of WorkflowOption and TaskOption.
func (c *Client) NewStandaloneDurableTask[I, O any](name string, fn func(ctx DurableContext, input I) (O, error), options ...StandaloneTaskOption) *StandaloneTask {
	if name == "" {
		panic("standalone durable task name cannot be empty")
	}

	workflowOptions, taskOptions := splitStandaloneOptions(name, options)

	workflow := c.NewWorkflow(name, workflowOptions...)
	task := workflow.NewDurableTask(name, fn, taskOptions...)

	return &StandaloneTask{
		workflow: workflow,
		task:     task,
	}
}
