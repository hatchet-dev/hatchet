package simple

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

type Result struct {
	AlwaysFails AlwaysFailsOutput `json:"always_fails"` // always_fails is the task name
}

// TODO type output..
type OnFailureOutput struct {
	FailureRan bool `json:"failure_ran"`
}

func Workflow(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[any, Result] {

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

	simple.Task(task.CreateOpts[any, Result]{
		Name: "always_fails",
		Fn: func(_ any, ctx worker.HatchetContext) (*Result, error) {
			return nil, errors.New("always fails")
		},
	})

	return simple
}
