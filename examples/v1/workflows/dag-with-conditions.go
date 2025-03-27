package v1_workflows

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type DagWithConditionsInput struct {
	Message string
}

type DagWithConditionsResult struct {
	Step1 SimpleOutput
	Step2 SimpleOutput
}

type conditionOpts = create.WorkflowTask[DagWithConditionsInput, DagWithConditionsResult]

func DagWithConditionsWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DagWithConditionsInput, DagWithConditionsResult] {

	simple := factory.NewWorkflow[DagWithConditionsInput, DagWithConditionsResult](
		create.WorkflowCreateOpts[DagWithConditionsInput]{
			Name: "simple-dag",
		},
		hatchet,
	)

	step1 := simple.Task(
		conditionOpts{
			Name: "Step1",
		}, func(ctx worker.HatchetContext, input DagWithConditionsInput) (interface{}, error) {
			return &SimpleOutput{
				Step: 1,
			}, nil
		},
	)

	simple.Task(
		conditionOpts{
			Name: "Step2",
			Parents: []create.NamedTask{
				step1,
			},
		}, func(ctx worker.HatchetContext, input DagWithConditionsInput) (interface{}, error) {

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
