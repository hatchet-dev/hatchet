package v1_workflows

import (
	"strings"
	"time"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type SimpleInput struct {
	Message string
}

type LowerOutput struct {
	TransformedMessage string
}

type SimpleResult struct {
	ToLower LowerOutput
}

func Simple(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[SimpleInput, SimpleResult] {

	simple := v1.WorkflowFactory[SimpleInput, SimpleResult](
		workflow.CreateOpts[SimpleInput]{
			Name: "simple",
		},
		hatchet,
	)

	simple.Task(
		task.CreateOpts[SimpleInput]{
			Name:             "ToLower",
			ExecutionTimeout: 10 * time.Second,
			Fn: func(input SimpleInput, ctx worker.HatchetContext) (*LowerOutput, error) {
				return &LowerOutput{
					TransformedMessage: strings.ToLower(input.Message),
				}, nil
			},
		},
	)

	return simple
}
