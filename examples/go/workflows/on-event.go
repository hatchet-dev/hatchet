package v1_workflows

import (
	"fmt"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type EventInput struct {
	Message string
}

type LowerTaskOutput struct {
	TransformedMessage string
}

type UpperTaskOutput struct {
	TransformedMessage string
}

// > Run workflow on event
const SimpleEvent = "simple-event:create"

func Lower(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[EventInput, LowerTaskOutput] {
	return factory.NewTask(
		create.StandaloneTask{
			Name: "lower",
			// 👀 Declare the event that will trigger the workflow
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

// > Accessing the filter payload
func accessFilterPayload(ctx worker.HatchetContext, input EventInput) (*LowerTaskOutput, error) {
	fmt.Println(ctx.FilterPayload())
	return &LowerTaskOutput{
		TransformedMessage: strings.ToLower(input.Message),
	}, nil
}

// > Declare with filter
func LowerWithFilter(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[EventInput, LowerTaskOutput] {
	return factory.NewTask(
		create.StandaloneTask{
			Name: "lower",
			// 👀 Declare the event that will trigger the workflow
			OnEvents: []string{SimpleEvent},
			DefaultFilters: []types.DefaultFilter{{
				Expression: "true",
				Scope:      "example-scope",
				Payload: map[string]interface{}{
					"main_character":       "Anna",
					"supporting_character": "Stiva",
					"location":             "Moscow"},
			}},
		}, accessFilterPayload,
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
