package v1_workflows

import (
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type SimpleOutput struct {
	Step int `json:"step"`
}

type DagResult struct {
	Step1 SimpleOutput `json:"step_1,omitempty"`
	Step2 SimpleOutput `json:"step_2,omitempty"`
}

func Workflow(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[any, DagResult] {

	simple := v1.WorkflowFactory[any, DagResult](
		workflow.CreateOpts{
			Name: "simple-dag",
		},
		hatchet,
	)

	step1 := simple.Task(task.CreateOpts[any, DagResult]{
		Name: "step_1",
		Fn: func(_ any, ctx worker.HatchetContext) (*DagResult, error) {
			return &DagResult{
				Step1: SimpleOutput{
					Step: 1,
				},
			}, nil
		},
	})

	simple.Task(task.CreateOpts[any, DagResult]{
		Name: "step_2",
		Parents: []*task.TaskDeclaration[any, DagResult]{
			step1,
		},
		Fn: func(_ any, ctx worker.HatchetContext) (*DagResult, error) {
			return &DagResult{
				Step2: SimpleOutput{
					Step: 2,
				},
			}, nil
		},
	})

	return simple
}
