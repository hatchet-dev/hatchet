package v1_workflows

import (
	"fmt"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type ChildInput struct {
	N int `json:"n"`
}

type ValueOutput struct {
	Value int `json:"value"`
}

type ChildResult struct {
	One ValueOutput `json:"one"`
}

type ParentInput struct {
	N int `json:"n"`
}

type SumOutput struct {
	Result int `json:"result"`
}

type ParentResult struct {
	Sum SumOutput `json:"sum"`
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
			Name: "one",
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

	child := Child(hatchet)
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

				// Create a channel to collect results and errors
				type childResponse struct {
					result *ChildResult
					err    error
				}
				responses := make(chan childResponse, input.N)

				// Launch child workflows in parallel
				for j := 0; j < input.N; j++ {
					go func() {
						childResult, err := child.RunAsChild(ctx, ChildInput{N: 1})
						responses <- childResponse{result: childResult, err: err}
					}()
				}

				// Collect all results
				var childResult *ChildResult
				for j := 0; j < input.N; j++ {
					response := <-responses
					if response.err != nil {
						return nil, response.err
					}
					childResult = response.result
				}

				fmt.Println("childResult", childResult.One.Value)

				sum += childResult.One.Value

				return &SumOutput{
					Result: 1000,
				}, nil
			},
		},
	)

	return parent
}
