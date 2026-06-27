package scheduled

import (
	"context"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Create
	scheduledRun, err := client.Schedules().Create(
		context.Background(),
		"scheduled",
		features.CreateScheduledRunTrigger{
			TriggerAt: time.Now().Add(1 * time.Minute),
			Input:     map[string]interface{}{"message": "Hello, World!"},
		},
	)
	if err != nil {
		log.Fatalf("failed to create scheduled run: %v", err)
	}
	// !!

	// Create two more scheduled runs to demonstrate bulk operations.
	scheduledRun2, err := client.Schedules().Create(
		context.Background(),
		"scheduled",
		features.CreateScheduledRunTrigger{
			TriggerAt: time.Now().Add(2 * time.Minute),
			Input:     map[string]interface{}{"message": "Hello, World! 2"},
		},
	)
	if err != nil {
		log.Fatalf("failed to create scheduled run: %v", err)
	}

	scheduledRun3, err := client.Schedules().Create(
		context.Background(),
		"scheduled",
		features.CreateScheduledRunTrigger{
			TriggerAt: time.Now().Add(3 * time.Minute),
			Input:     map[string]interface{}{"message": "Hello, World! 3"},
		},
	)
	if err != nil {
		log.Fatalf("failed to create scheduled run: %v", err)
	}

	// > Update
	updatedRun, err := client.Schedules().Update(
		context.Background(),
		scheduledRun.Metadata.Id,
		features.CreateScheduledRunTrigger{
			TriggerAt: time.Now().Add(5 * time.Minute),
		},
	)
	if err != nil {
		log.Fatalf("failed to update scheduled run: %v", err)
	}
	// !!

	// > BulkUpdate
	bulkUpdateResp, err := client.Schedules().BulkUpdate(
		context.Background(),
		[]features.BulkUpdateScheduledRunItem{
			{
				ScheduledRunId: scheduledRun2.Metadata.Id,
				TriggerAt:      time.Now().Add(10 * time.Minute),
			},
			{
				ScheduledRunId: scheduledRun3.Metadata.Id,
				TriggerAt:      time.Now().Add(15 * time.Minute),
			},
		},
	)
	if err != nil {
		log.Fatalf("failed to bulk update scheduled runs: %v", err)
	}
	// !!

	// > Delete
	err = client.Schedules().Delete(
		context.Background(),
		updatedRun.Metadata.Id,
	)
	if err != nil {
		log.Fatalf("failed to delete scheduled run: %v", err)
	}
	// !!

	// > BulkDelete
	bulkDeleteResp, err := client.Schedules().BulkDelete(
		context.Background(),
		[]string{scheduledRun2.Metadata.Id, scheduledRun3.Metadata.Id},
		nil,
	)
	if err != nil {
		log.Fatalf("failed to bulk delete scheduled runs: %v", err)
	}
	// !!

	// > List
	scheduledRuns, err := client.Schedules().List(
		context.Background(),
		rest.WorkflowScheduledListParams{},
	)
	if err != nil {
		log.Fatalf("failed to list scheduled runs: %v", err)
	}
	// !!

	_ = bulkUpdateResp
	_ = bulkDeleteResp
	_ = scheduledRuns
}
