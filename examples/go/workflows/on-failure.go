package v1_workflows

import (
	"errors"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
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

func OnFailure(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[any, OnFailureSuccessResult] {

	simple := factory.NewWorkflow[any, OnFailureSuccessResult](
		create.WorkflowCreateOpts[any]{
			Name: "on-failure",
		},
		hatchet,
	)

	simple.Task(
		create.WorkflowTask[any, OnFailureSuccessResult]{
			Name: "AlwaysFails",
		},
		func(ctx worker.HatchetContext, _ any) (interface{}, error) {
			return &AlwaysFailsOutput{
				TransformedMessage: "always fails",
			}, errors.New("always fails")
		},
	)

	simple.OnFailure(
		create.WorkflowOnFailureTask[any, OnFailureSuccessResult]{},
		func(ctx worker.HatchetContext, _ any) (interface{}, error) {
			return &OnFailureOutput{
				FailureRan: true,
			}, nil
		},
	)

	return simple
}
