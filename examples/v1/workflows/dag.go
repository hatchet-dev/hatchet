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
	Step1 SimpleOutput `json:"Step1,omitempty"`
	Step2 SimpleOutput `json:"Step2,omitempty"`
}

func Workflow(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[any, DagResult] {

	simple := v1.WorkflowFactory[any, DagResult](
		workflow.CreateOpts{
			Name: "simple-dag",
		},
		hatchet,
	)

	simple.Task(
		task.CreateOpts[any]{
			Name: "Step1",
		},
		func(_ any, ctx worker.HatchetContext) (*SimpleOutput, error) {
			return &SimpleOutput{
				Step: 1,
			}, nil
		})

	simple.Task(
		task.CreateOpts[any]{
			Name: "Step2",
		},
		func(_ any, ctx worker.HatchetContext) (*SimpleOutput, error) {
			return &SimpleOutput{
				Step: 2,
			}, nil

		})

	return simple
}
