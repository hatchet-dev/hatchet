package v1_workflows

import (
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type ChildInput struct {
	N int
}

type ValueOutput struct {
	Value int
}

type ChildResult struct {
	Value ValueOutput
}

type ParentInput struct {
	N int
}

type SumOutput struct {
	Result int
}

type ParentResult struct {
	Sum SumOutput
}

func Child(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[ChildInput, ChildResult] {
	child := v1.WorkflowFactory[ChildInput, ChildResult](
		workflow.CreateOpts[ChildInput]{
			Name: "child",
		},
		hatchet,
	)

	child.Task(
		task.CreateOpts[ChildInput]{
			Name: "value",
			Fn: func(input ChildInput, ctx worker.HatchetContext) (*ValueOutput, error) {
				return &ValueOutput{
					Value: input.N,
				}, nil
			},
		},
	)

	return child
}

func Parent(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[ParentInput, ParentResult] {
	parent := v1.WorkflowFactory[ParentInput, ParentResult](
		workflow.CreateOpts[ParentInput]{
			Name: "parent",
		},
		hatchet,
	)

	parent.Task(
		task.CreateOpts[ParentInput]{
			Name: "sum",
			Fn: func(input ParentInput, ctx worker.HatchetContext) (*SumOutput, error) {
				sum := 0

				for i := 0; i < input.N; i++ {
					workflow, err := ctx.SpawnWorkflow("child", ChildInput{N: i}, nil)
					if err != nil {
						return nil, err
					}

					result, err := workflow.Result()
					if err != nil {
						return nil, err
					}

					var childResult ChildResult
					err = result.StepOutput("value", &childResult.Value)
					if err != nil {
						return nil, err
					}

					sum += childResult.Value.Value
				}

				return &SumOutput{
					Result: sum,
				}, nil
			},
		},
	)

	return parent
}
