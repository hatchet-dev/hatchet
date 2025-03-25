package v1_workflows

import (
	"sync"

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
	Sum *SumOutput `json:"sum,omitempty"`
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

				// Use a WaitGroup to coordinate parallel child workflows
				var wg sync.WaitGroup
				var mu sync.Mutex
				var firstErr error

				// Launch child workflows in parallel
				results := make([]*ChildResult, 0, input.N)
				wg.Add(input.N)
				for j := 0; j < input.N; j++ {
					go func(index int) {
						defer wg.Done()
						result, err := child.RunAsChild(ctx, ChildInput{N: 1})

						mu.Lock()
						defer mu.Unlock()

						if err != nil && firstErr == nil {
							firstErr = err
							return
						}

						if firstErr == nil {
							results = append(results, result)
						}
					}(j)
				}

				// Wait for all goroutines to complete
				wg.Wait()

				// Check if any errors occurred
				if firstErr != nil {
					return nil, firstErr
				}

				// Sum results from all children
				for _, result := range results {
					sum += result.One.Value
				}

				return &SumOutput{
					Result: sum,
				}, nil
			},
		},
	)

	return parent
}
