package v1_workflows

import (
	"strings"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
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

type LowerResult struct {
	Lower LowerTaskOutput
}

type UpperTaskOutput struct {
	TransformedMessage string
}

type UpperResult struct {
	Upper UpperTaskOutput
}

func Lower(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[EventInput, LowerResult] {
	lower := v1.WorkflowFactory[EventInput, LowerResult](
		workflow.CreateOpts[EventInput]{
			Name:     "lower",
			OnEvents: []string{SimpleEvent},
		},
		hatchet,
	)

	lower.Task(
		task.CreateOpts[EventInput]{
			Name: "lower",
			Fn: func(input EventInput, ctx worker.HatchetContext) (*LowerTaskOutput, error) {
				return &LowerTaskOutput{
					TransformedMessage: strings.ToLower(input.Message),
				}, nil
			},
		},
	)

	return lower
}

func Upper(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[EventInput, UpperResult] {
	upper := v1.WorkflowFactory[EventInput, UpperResult](
		workflow.CreateOpts[EventInput]{
			Name:     "upper",
			OnEvents: []string{SimpleEvent},
		},
		hatchet,
	)

	upper.Task(
		task.CreateOpts[EventInput]{
			Name: "upper",
			Fn: func(input EventInput, ctx worker.HatchetContext) (*UpperTaskOutput, error) {
				return &UpperTaskOutput{
					TransformedMessage: strings.ToUpper(input.Message),
				}, nil
			},
		},
	)

	return upper
}
