package v1_workflows

import (
	"errors"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type AlwaysFailsOutput struct {
	TransformedMessage string `json:"message"`
}

// TODO type output..
type OnFailureOutput struct {
	FailureRan bool `json:"FailureRan"`
}

func OnFailure(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[any, Result] {

	simple := v1.WorkflowFactory[any, Result](
		workflow.CreateOpts{
			Name: "on-failure",
			OnFailureTask: &task.OnFailureTaskDeclaration[any, any]{
				Fn: func(_ any, ctx worker.HatchetContext) (*any, error) {
					return nil, nil
				},
			},
		},
		hatchet,
	)

	simple.Task(
		task.CreateOpts[any]{
			Name: "AlwaysFails",
		},
		func(_ any, ctx worker.HatchetContext) (*AlwaysFailsOutput, error) {
			return &AlwaysFailsOutput{
				TransformedMessage: "always fails",
			}, errors.New("always fails")
		},
	)

	return simple
}
