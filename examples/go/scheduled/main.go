package scheduled

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type SimpleInput struct {
	Message string `json:"message"`
}

type SimpleOutput struct {
	Result string `json:"result"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	simple := client.NewStandaloneTask("simple", func(ctx hatchet.Context, input SimpleInput) (SimpleOutput, error) {
		return SimpleOutput{
			Result: "Processed: " + input.Message,
		}, nil
	})

	// > Create
	tomorrow := time.Now().UTC().AddDate(0, 0, 1)
	tomorrowNoon := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 12, 0, 0, 0, time.UTC)

	scheduledRun, err := simple.Schedule(context.Background(), tomorrowNoon, SimpleInput{Message: "Hello, World!"})
	if err != nil {
		log.Fatalf("failed to create scheduled run: %v", err)
	}

	scheduledRunId := scheduledRun.GetScheduledWorkflows()[0].GetId()

	// > Delete
	client.Schedules().Delete(
		context.Background(),
		scheduledRunId,
	)

	// > List
	client.Schedules().List(
		context.Background(),
		rest.WorkflowScheduledListParams{},
	)

	// > Reschedule
	client.Schedules().Update(
		context.Background(),
		scheduledRunId,
		rest.UpdateScheduledWorkflowRunRequest{
			TriggerAt: time.Now().UTC().Add(10 * time.Second),
		},
	)

	scheduledRunIds := []types.UUID{types.UUID(uuid.MustParse(scheduledRunId))}

	// > Bulk Delete
	client.Schedules().BulkDelete(
		context.Background(),
		rest.ScheduledWorkflowsBulkDeleteRequest{
			ScheduledWorkflowRunIds: &scheduledRunIds,
		},
	)

	scheduledRunIdUUID := types.UUID(uuid.MustParse(scheduledRunId))

	// > Reschedule
	client.Schedules().Update(
		context.Background(),
		scheduledRunId,
		rest.UpdateScheduledWorkflowRunRequest{
			TriggerAt: time.Now().UTC().Add(10 * time.Second),
		},
	)

	// > Bulk Update
	client.Schedules().BulkUpdate(
		context.Background(),
		rest.ScheduledWorkflowsBulkUpdateRequest{
			Updates: []rest.ScheduledWorkflowsBulkUpdateItem{
				{Id: scheduledRunIdUUID, TriggerAt: time.Now().UTC().Add(10 * time.Second)},
			},
		},
	)

	worker, err := client.NewWorker("scheduled-worker", hatchet.WithWorkflows(simple))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	if err := worker.StartBlocking(context.Background()); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
