package v1_workflows

import (
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

const SimpleEvent = "simple-event:create"

type EventInput struct {
	Message string
}

type LowerTaskOutput struct {
	TransformedMessage string
}

type UpperTaskOutput struct {
	TransformedMessage string
}

func Lower(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[EventInput, LowerTaskOutput] {
	return factory.NewTask(
		create.StandaloneTask{
			Name:     "lower",
			OnEvents: []string{SimpleEvent},
		}, func(ctx worker.HatchetContext, input EventInput) (*LowerTaskOutput, error) {
			// Transform the input message to lowercase
			return &LowerTaskOutput{
				TransformedMessage: strings.ToLower(input.Message),
			}, nil
		},
		hatchet,
	)
}

func Upper(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[EventInput, UpperTaskOutput] {
	return factory.NewTask(
		create.StandaloneTask{
			Name:     "upper",
			OnEvents: []string{SimpleEvent},
		},
		func(ctx worker.HatchetContext, input EventInput) (*UpperTaskOutput, error) {
			return &UpperTaskOutput{
				TransformedMessage: strings.ToUpper(input.Message),
			}, nil
		},
		hatchet,
	)
}
