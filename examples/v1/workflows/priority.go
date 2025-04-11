package v1_workflows

import (
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type PriorityInput struct {
	UserId string `json:"userId"`
}

type PriorityOutput struct {
	TransformedMessage string `json:"TransformedMessage"`
}

type Result struct {
	Step PriorityOutput
}

// ❓ Static Rate Limit
func Priority(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[PriorityInput, Result] {
	// Create a standalone task that transforms a message

	defaultPriority := int32(1)

	workflow := factory.NewWorkflow[PriorityInput, Result](
		create.WorkflowCreateOpts[PriorityInput]{
			Name:            "priority",
			DefaultPriority: &defaultPriority,
		},
		hatchet,
	)

	// ❓ Defining a Task
	workflow.Task(
		create.WorkflowTask[PriorityInput, Result]{
			Name: "step",
		}, func(ctx worker.HatchetContext, input PriorityInput) (interface{}, error) {
			return &PriorityOutput{
				TransformedMessage: input.UserId,
			}, nil
		},
	)
	// ‼️
	return workflow
}

// !!
