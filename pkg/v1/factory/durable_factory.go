package factory

import (
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// NewDurableTask creates a durable task that is implemented as a simple workflow with a single task.
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
//	    Fn: func(input SimpleInput, ctx worker.DurableHatchetContext) (*SimpleOutput, error) {
//	        return &SimpleOutput{
//	            TransformedMessage: strings.ToLower(input.Message),
//	        }, nil
//	    },
//	}, &client)
func NewDurableTask[I, O any](opts create.StandaloneTask, fn func(input I, ctx worker.DurableHatchetContext) (*O, error), client v1.HatchetClient) workflow.WorkflowDeclaration[I, O] {
	var v0 v0Client.Client
	if client != nil {
		v0 = client.V0()
	}

	// Create a workflow with the same name as the task
	workflowOpts := create.WorkflowCreateOpts[I]{
		Name:        opts.Name,
		Version:     opts.Version,
		Description: opts.Description,
		OnEvents:    opts.OnEvents,
		OnCron:      opts.OnCron,
		OutputKey:   &opts.Name,
		// OnFailureTask:  opts.OnFailureTask,
		// StickyStrategy: opts.StickyStrategy,
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
	}

	fixedFn := func(input I, ctx worker.DurableHatchetContext) (interface{}, error) {
		return fn(input, ctx)
	}

	// Register the task within the workflow
	workflowDecl.DurableTask(taskOpts, fixedFn)

	return workflowDecl
}
