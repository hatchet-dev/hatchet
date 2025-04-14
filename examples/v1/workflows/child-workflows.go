package v1_workflows

import (
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type ChildInput struct {
	N int `json:"n"`
}

type ValueOutput struct {
	Value int `json:"value"`
}

type ParentInput struct {
	N int `json:"n"`
}

type SumOutput struct {
	Result int `json:"result"`
}

func Child(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ChildInput, ValueOutput] {
	child := factory.NewTask(
		create.StandaloneTask{
			Name: "child",
		}, func(ctx worker.HatchetContext, input ChildInput) (*ValueOutput, error) {
			return &ValueOutput{
				Value: input.N,
			}, nil
		},
		hatchet,
	)

	return child
}

func Parent(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ParentInput, SumOutput] {

	child := Child(hatchet)
	parent := factory.NewTask(
		create.StandaloneTask{
			Name: "parent",
		}, func(ctx worker.HatchetContext, input ParentInput) (*SumOutput, error) {

			sum := 0

			// Launch child workflows in parallel
			results := make([]*ValueOutput, 0, input.N)
			for j := 0; j < input.N; j++ {
				result, err := child.RunAsChild(ctx, ChildInput{N: j}, workflow.RunAsChildOpts{})

				if err != nil {
					// firstErr = err
					return nil, err
				}

				results = append(results, result)

			}

			// Sum results from all children
			for _, result := range results {
				sum += result.Value
			}

			return &SumOutput{
				Result: sum,
			}, nil
		},
		hatchet,
	)

	return parent
}
