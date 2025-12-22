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

	// > Delete
	err = client.Schedules().Delete(
		context.Background(),
		scheduledRun.Metadata.Id,
	)
	if err != nil {
		log.Fatalf("failed to delete scheduled run: %v", err)
	}

	// > List
	scheduledRuns, err := client.Schedules().List(
		context.Background(),
		rest.WorkflowScheduledListParams{},
	)
	if err != nil {
		log.Fatalf("failed to list scheduled runs: %v", err)
	}

	_ = scheduledRuns
}
