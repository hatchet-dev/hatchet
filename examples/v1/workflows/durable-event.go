package v1_workflows

import (
	"context"
	"time"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type DurableEventInput struct {
	Message string
}

type EventData struct {
	Message string
}

type EventOutput struct {
	Data EventData
}

type DurableEventOutput struct {
	Event EventOutput
}

func DurableEvent(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DurableEventInput, DurableEventOutput] {

	durableEventWorkflow := v1.WorkflowFactory[DurableEventInput, DurableEventOutput](
		workflow.CreateOpts[DurableEventInput]{
			Name: "durable-event",
		},
		&hatchet,
	)

	durableEventWorkflow.DurableTask(
		task.CreateOpts[DurableEventInput]{
			Name: "Event",
			Fn: func(input DurableEventInput, ctx worker.DurableHatchetContext) (*EventOutput, error) {
				eventData, err := ctx.WaitForEvent("user:update", "")

				if err != nil {
					return nil, err
				}

				v := EventData{}
				err = eventData.Unmarshal(&v)

				if err != nil {
					return nil, err
				}

				return &EventOutput{
					Data: v,
				}, nil
			},
		},
	)

	go func() {
		time.Sleep(45 * time.Second)

		durableEventWorkflow.RunNoWait(DurableEventInput{
			Message: "Hello, World!",
		})

		time.Sleep(10 * time.Second)

		hatchet.V0().Event().Push(context.Background(), "user:update", EventData{
			Message: "User updated!",
		})
	}()

	return durableEventWorkflow
}
