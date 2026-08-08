//go:build !go1.27

// Reflection-based (fn any) standalone task constructors for Go 1.26 and earlier. On
// Go 1.27+, standalone_constructors_go127.go provides the same method names with typed,
// reflection-free signatures. The two files are mutually exclusive via the go1.27 build tag.

package hatchet

// NewStandaloneTask creates a standalone task that can be triggered independently.
// This is a specialized workflow containing only one task, making it easier to create
// simple single-task workflows without the workflow boilerplate.
//
// The function parameter must have the signature:
//
//	func(ctx hatchet.Context, input any) (any, error)
//
// Function signatures are validated at runtime using reflection.
//
// Options can be any combination of WorkflowOption and TaskOption.
func (c *Client) NewStandaloneTask(name string, fn any, options ...StandaloneTaskOption) *StandaloneTask {
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
// independently. This is a specialized workflow containing only one durable task.
//
// The function parameter must have the signature:
//
//	func(ctx hatchet.DurableContext, input any) (any, error)
//
// Function signatures are validated at runtime using reflection.
//
// Options can be any combination of WorkflowOption and TaskOption.
func (c *Client) NewStandaloneDurableTask(name string, fn any, options ...StandaloneTaskOption) *StandaloneTask {
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
