package simple

import (
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type SimpleOutput struct {
	Step int `json:"step"`
}

type Result struct {
	Step1 SimpleOutput `json:"step_1,omitempty"`
	Step2 SimpleOutput `json:"step_2,omitempty"`
}

func Workflow(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[any, Result] {

	simple := v1.WorkflowFactory[any, Result](
		workflow.CreateOpts{
			Name: "simple-dag",
		},
		hatchet,
	)

	step1 := simple.Task(task.CreateOpts[any, Result]{
		Name: "step_1",
		Fn: func(_ any, ctx worker.HatchetContext) (*Result, error) {
			return &Result{
				Step1: SimpleOutput{
					Step: 1,
				},
			}, nil
		},
	})

	simple.Task(task.CreateOpts[any, Result]{
		Name: "step_2",
		Parents: []*task.TaskDeclaration[any, Result]{
			step1,
		},
		Fn: func(_ any, ctx worker.HatchetContext) (*Result, error) {
			return &Result{
				Step2: SimpleOutput{
					Step: 2,
				},
			}, nil
		},
	})

	return simple
}
