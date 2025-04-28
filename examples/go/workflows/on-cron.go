package v1_workflows

import (
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type OnCronInput struct {
	Message string `json:"Message"`
}

type JobResult struct {
	TransformedMessage string `json:"TransformedMessage"`
}

type OnCronOutput struct {
	Job JobResult `json:"job"`
}

// > Workflow Definition Cron Trigger
func OnCron(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[OnCronInput, OnCronOutput] {
	// Create a standalone task that transforms a message
	cronTask := factory.NewTask(
		create.StandaloneTask{
			Name: "on-cron-task",
			// 👀 add a cron expression
			OnCron: []string{"0 0 * * *"}, // Run every day at midnight
		},
		func(ctx worker.HatchetContext, input OnCronInput) (*OnCronOutput, error) {
			return &OnCronOutput{
				Job: JobResult{
					TransformedMessage: strings.ToLower(input.Message),
				},
			}, nil
		},
		hatchet,
	)

	return cronTask
}

// !!
