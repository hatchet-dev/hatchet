package v1_workflows

import (
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type DurableSleepInput struct {
	Message string
}

type DurableSleepOutput struct {
	TransformedMessage string
}

func DurableSleep(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DurableSleepInput, DurableSleepOutput] {
	// ctx as first param of NewTask
	simple := factory.NewDurableTask(
		create.StandaloneTask{
			Name: "durable-sleep",
		},
		func(ctx worker.DurableHatchetContext, input DurableSleepInput) (*DurableSleepOutput, error) {
			_, err := ctx.SleepFor(10 * time.Second)

			if err != nil {
				return nil, err
			}

			return &DurableSleepOutput{
				TransformedMessage: strings.ToLower(input.Message),
			}, nil
		},
		hatchet,
	)

	return simple
}
