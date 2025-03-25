package v1_workflows

import (
	"errors"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type AlwaysFailsOutput struct {
	TransformedMessage string
}

type OnFailureOutput struct {
	FailureRan bool
}

type OnFailureSuccessResult struct {
	AlwaysFails AlwaysFailsOutput
}

func OnFailure(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[any, OnFailureSuccessResult] {

	simple := v1.WorkflowFactory[any, OnFailureSuccessResult](
		workflow.CreateOpts[any]{
			Name: "on-failure",
			OnFailureTask: &task.OnFailureTaskDeclaration[any]{
				Fn: func(_ any, ctx worker.HatchetContext) (*OnFailureOutput, error) {
					return &OnFailureOutput{
						FailureRan: true,
					}, nil
				},
			},
		},
		hatchet,
	)

	simple.Task(
		task.CreateOpts[any]{
			Name: "AlwaysFails",
			Fn: func(_ any, ctx worker.HatchetContext) (*AlwaysFailsOutput, error) {
				return &AlwaysFailsOutput{
					TransformedMessage: "always fails",
				}, errors.New("always fails")
			},
		},
	)

	return simple
}
