package factory

import (
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// NewTask creates a standalone task that is implemented as a simple workflow with a single task.
// It provides proper type inference for the input and output types.
//
// Example:
//
//	type SimpleInput struct {
//	    Message string
//	}
//
//	type SimpleOutput struct {
//	    TransformedMessage string
//	}
//
//	simpleTask := v1.NewTask(v1.TaskCreateOpts[SimpleInput, SimpleOutput]{
//	    Name: "simple-task",
//	    Fn: func(input SimpleInput, ctx worker.HatchetContext) (*SimpleOutput, error) {
//	        return &SimpleOutput{
//	            TransformedMessage: strings.ToLower(input.Message),
//	        }, nil
//	    },
//	}, &client)
func NewTask[I, O any](opts create.StandaloneTask, fn func(ctx worker.HatchetContext, input I) (*O, error), client v1.HatchetClient) workflow.WorkflowDeclaration[I, O] {
	var v0 v0Client.Client
	if client != nil {
		v0 = client.V0()
	}

	// Create a workflow with the same name as the task
	workflowOpts := create.WorkflowCreateOpts[I]{
		Name:            opts.Name,
		Version:         opts.Version,
		Description:     opts.Description,
		OnEvents:        opts.OnEvents,
		OnCron:          opts.OnCron,
		OutputKey:       &opts.Name,
		DefaultPriority: opts.DefaultPriority,
	}

	// Create the workflow
	workflowDecl := workflow.NewWorkflowDeclaration[I, O](workflowOpts, v0)

	// Convert to task.CreateOpts
	taskOpts := create.WorkflowTask[I, O]{
		Name:                   opts.Name,
		ExecutionTimeout:       opts.ExecutionTimeout,
		ScheduleTimeout:        opts.ScheduleTimeout,
		Retries:                opts.Retries,
		RetryBackoffFactor:     opts.RetryBackoffFactor,
		RetryMaxBackoffSeconds: opts.RetryMaxBackoffSeconds,
		RateLimits:             opts.RateLimits,
		WorkerLabels:           opts.WorkerLabels,
		Concurrency:            opts.Concurrency,
		DefaultPriority:        opts.DefaultPriority,
	}

	fixedFn := func(ctx worker.HatchetContext, input I) (interface{}, error) {
		return fn(ctx, input)
	}

	// Register the task within the workflow
	workflowDecl.Task(taskOpts, fixedFn)

	return workflowDecl
}
