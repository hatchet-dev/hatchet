package v1_workflows

import (
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

func DagWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DagInput, DagResult] {
	// > Declaring a Workflow
	simple := factory.NewWorkflow[DagInput, DagResult](
		create.WorkflowCreateOpts[DagInput]{
			Name: "simple-dag",

		},
		hatchet,
	)

	// > Defining a Task
	simple.Task(
		create.WorkflowTask[DagInput, DagResult]{
			Name: "step",
		}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {
			return &SimpleOutput{
				Step: 1,
			}, nil
		},
	)

	// > Adding a Task with a parent
	step1 := simple.Task(
		create.WorkflowTask[DagInput, DagResult]{
			Name: "step-1",
		}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {
			return &SimpleOutput{
				Step: 1,
			}, nil
		},
	)

	simple.Task(
		create.WorkflowTask[DagInput, DagResult]{
			Name: "step-2",
			Parents: []create.NamedTask{
				step1,
			},
		}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {
			// Get the output of the parent task
			var step1Output SimpleOutput
			err := ctx.ParentOutput(step1, &step1Output)
			if err != nil {
				return nil, err
			}

			return &SimpleOutput{
				Step: 2,
			}, nil
		},
	)

	return simple
}
