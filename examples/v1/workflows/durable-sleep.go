package v1_workflows

import (
	"strings"
	"time"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type DurableSleepInput struct {
	Message string
}

type SleepOutput struct {
	TransformedMessage string
}

type DurableSleepOutput struct {
	Sleep SleepOutput
}

func DurableSleep(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[DurableSleepInput, DurableSleepOutput] {

	simple := v1.WorkflowFactory[DurableSleepInput, DurableSleepOutput](
		workflow.CreateOpts[DurableSleepInput]{
			Name: "durable-sleep",
		},
		hatchet,
	)

	simple.Task(
		task.CreateOpts[DurableSleepInput]{
			Name: "Non-Durable-Sleep",
			Fn: func(input DurableSleepInput, ctx worker.DurableHatchetContext) (*SleepOutput, error) {

				_, err := ctx.SleepFor(time.Minute)

				if err != nil {
					return nil, err
				}

				return &SleepOutput{
					TransformedMessage: strings.ToLower(input.Message),
				}, nil
			},
		},
	)

	simple.DurableTask(
		task.CreateOpts[DurableSleepInput]{
			Name: "Sleep",
			Fn: func(input DurableSleepInput, ctx worker.DurableHatchetContext) (*SleepOutput, error) {

				_, err := ctx.SleepFor(time.Minute)

				if err != nil {
					return nil, err
				}

				return &SleepOutput{
					TransformedMessage: strings.ToLower(input.Message),
				}, nil
			},
		},
	)

	return simple
}
