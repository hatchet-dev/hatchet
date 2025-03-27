package v1_workflows

import (
	"sync"

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

			// Use a WaitGroup to coordinate parallel child workflows
			var wg sync.WaitGroup
			var mu sync.Mutex
			var firstErr error

			// Launch child workflows in parallel
			results := make([]*ValueOutput, 0, input.N)
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
