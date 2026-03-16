package main

import (
	"context"
	"log"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

// > Step 02 Schedule One Time
func scheduleOneTime(client *hatchet.Client) {
	runAt := time.Now().Add(1 * time.Hour)
	_, err := client.Schedules().Create(context.Background(), "run-scheduled-job", features.CreateScheduledRunTrigger{
		TriggerAt: runAt,
		Input:     map[string]interface{}{},
	})
	if err != nil {
		log.Printf("failed to schedule: %v", err)
	}
}

