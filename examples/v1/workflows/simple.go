package v1_workflows

import (
	"fmt"
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

func Simple(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[Input, Result] {

	simple := v1.WorkflowFactory[Input, Result](
		workflow.CreateOpts{
			Name: "simple",
		},
		hatchet,
	)

	simple.Task(task.CreateOpts[Input, Result]{
		Name: "to_lower",
		Fn: func(input Input, ctx worker.HatchetContext) (*Result, error) {
			// TODO: this is a hack to get the result out of the function

			fmt.Println("input", input)
			result := &Result{
				ToLower: LowerOutput{
					TransformedMessage: strings.ToLower(input.Message),
				},
			}

			return result, nil
		},
	})

	return simple
}
