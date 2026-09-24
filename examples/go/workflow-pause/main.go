package main

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

	ctx := context.Background()

	// > Pause a workflow
	_, err = client.Workflows().Pause(ctx, "pausable-workflow", features.PauseWorkflowOpts{
		QueueTTL:                  24 * time.Hour,
		CronRunQueueBehavior:      rest.DROP,
		ScheduledRunQueueBehavior: rest.QUEUE,
	})
	if err != nil {
		log.Fatalf("failed to pause workflow: %v", err)
	}

	// > Unpause a workflow
	_, err = client.Workflows().Unpause(ctx, "pausable-workflow")
	if err != nil {
		log.Fatalf("failed to unpause workflow: %v", err)
	}
}
