package v1_workflows

import (
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type SimpleInput struct {
	Message string
}
type SimpleResult struct {
	TransformedMessage string
}

func Simple(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[SimpleInput, SimpleResult] {

	// Create a simple standalone task using the task factory
	// Note the use of typed generics for both input and output
	simple := factory.NewTask(
		create.StandaloneTask{
			Name: "simple-task",
		}, func(ctx worker.HatchetContext, input SimpleInput) (*SimpleResult, error) {
			// Transform the input message to lowercase
			return &SimpleResult{
				TransformedMessage: strings.ToLower(input.Message),
			}, nil
		},
		hatchet,
	)

	return simple
}
