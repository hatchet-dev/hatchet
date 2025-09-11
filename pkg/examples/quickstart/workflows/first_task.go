package workflows

import (
	"fmt"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type SimpleInput struct {
	Message string `json:"message"`
}

type LowerOutput struct {
	TransformedMessage string `json:"transformed_message"`
}

type SimpleResult struct {
	ToLower LowerOutput
}

func FirstTask(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[SimpleInput, SimpleResult] {
	simple := factory.NewWorkflow[SimpleInput, SimpleResult](
		create.WorkflowCreateOpts[SimpleInput]{
			Name: "first-task",
		},
		hatchet,
	)

	simple.Task(
		create.WorkflowTask[SimpleInput, SimpleResult]{
			Name: "first-task",
		},
		func(ctx worker.HatchetContext, input SimpleInput) (any, error) {
			fmt.Println("first-task task called")
			return &LowerOutput{
				TransformedMessage: strings.ToLower(input.Message),
			}, nil
		},
	)

	return simple
}
