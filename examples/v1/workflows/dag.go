package v1_workflows

import (
	"fmt"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type SimpleOutput struct {
	Step int
}

type DagResult struct {
	Step1 SimpleOutput
	Step2 SimpleOutput
}

func DagWorkflow(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[any, DagResult] {

	simple := v1.WorkflowFactory[any, DagResult](
		workflow.CreateOpts[any]{
			Name: "simple-dag",
		},
		hatchet,
	)

	step1 := simple.Task(
		task.CreateOpts[any]{
			Name: "Step1",
			Fn: func(_ any, ctx worker.HatchetContext) (*SimpleOutput, error) {
				return &SimpleOutput{
					Step: 1,
				}, nil
			},
		},
	)

	simple.Task(
		task.CreateOpts[any]{
			Name: "Step2",
			Parents: []*task.NamedTask{
				&step1.NamedTask, // TODO improve this
			},
			Fn: func(_ any, ctx worker.HatchetContext) (*SimpleOutput, error) {

				var step1Output SimpleOutput
				err := ctx.ParentOutput(step1, &step1Output)
				if err != nil {
					return nil, err
				}

				fmt.Println(step1Output.Step)

				return &SimpleOutput{
					Step: 2,
				}, nil
			},
		},
	)

	return simple
}
