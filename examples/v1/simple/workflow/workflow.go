package simple

import (
	"strings"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type Input struct {
	Message string `json:"message"`
}

type LowerOutput struct {
	TransformedMessage string `json:"message"`
}

type ReverseOutput struct {
	TransformedMessage string `json:"message"`
}

type Result struct {
	ToLower LowerOutput   `json:"to_lower"` // to_lower is the task name
	Reverse ReverseOutput `json:"reverse"`  // reverse is the task name
}

func SimpleWorkflow(hatchet *v1.HatchetClient) (workflow.WorkflowDeclaration[Input, Result], error) {

	simple := v1.WorkflowFactory[Input, Result](
		workflow.CreateOpts{
			Name: "simple",
		},
		hatchet,
	)

	lower := simple.Task(task.CreateOpts[Input, Result]{
		Name: "to_lower",
		Fn: func(input Input, ctx worker.HatchetContext) (*Result, error) {
			// TODO: this is a hack to get the result out of the function
			result := &Result{
				ToLower: LowerOutput{
					TransformedMessage: strings.ToLower(input.Message),
				},
			}

			return result, nil
		},
	})

	simple.Task(task.CreateOpts[Input, Result]{
		Name:    "reverse",
		Parents: simple.WithParents(lower),
		Fn: func(input Input, ctx worker.HatchetContext) (*Result, error) {
			reversed := ""
			for _, char := range input.Message {
				reversed = string(char) + reversed
			}

			result := &Result{
				Reverse: ReverseOutput{
					TransformedMessage: reversed,
				},
			}

			return result, nil
		},
	})

	return simple, nil
}
