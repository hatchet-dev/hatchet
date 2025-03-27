package v1_workflows

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type DagInput struct {
	Message string
}

type SimpleOutput struct {
	Step int
}

type DagResult struct {
	Step1 SimpleOutput
	Step2 SimpleOutput
}

type taskOpts = create.WorkflowTask[DagInput, DagResult]

func DagWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DagInput, DagResult] {

	simple := factory.NewWorkflow[DagInput, DagResult](
		create.WorkflowCreateOpts[DagInput]{
			Name: "simple-dag",
		},
		hatchet,
	)

	step1 := simple.Task(
		taskOpts{
			Name: "Step1",
		}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {
			return &SimpleOutput{
				Step: 1,
			}, nil
		},
	)

	simple.Task(
		taskOpts{
			Name: "Step2",
			Parents: []create.NamedTask{
				step1,
			},
		}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {

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
	)

	return simple
}
