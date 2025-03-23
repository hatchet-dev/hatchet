package v1_workflows

import (
	"fmt"
	"strings"
	"time"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type SimpleInput struct {
	Message string `json:"message"`
}

type LowerOutput struct {
	TransformedMessage string `json:"message"`
}

type SimpleResult struct {
	ToLower LowerOutput `json:"ToLower"` // to_lower is the task name
}

func Simple(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[SimpleInput, SimpleResult] {

	simple := v1.WorkflowFactory[SimpleInput, SimpleResult](
		workflow.CreateOpts[SimpleInput]{
			Name: "simple",
			TaskDefaults: &task.TaskDefaults{
				ExecutionTimeout:       10 * time.Second,
				Retries:                3,
				RetryBackoffFactor:     2,
				RetryMaxBackoffSeconds: 60,
			},
		},
		hatchet,
	)

	toLower := simple.Task(
		task.CreateOpts[SimpleInput]{
			Name:             "ToLower", // field name in Result
			ExecutionTimeout: 10 * time.Second,
			Fn: func(input SimpleInput, ctx worker.HatchetContext) (*LowerOutput, error) {
				return &LowerOutput{
					TransformedMessage: strings.ToLower(input.Message),
				}, nil
			},
		},
	)

	fmt.Println(toLower.Name)

	return simple
}
