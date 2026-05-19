package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Define Cron Task
	task := client.NewStandaloneTask("run-scheduled-job", func(ctx hatchet.Context, input map[string]interface{}) (map[string]string, error) {
		return map[string]string{"status": "completed", "job": "maintenance"}, nil
	}, hatchet.WithWorkflowCron("0 * * * *"))

	// > Step 03 Run Worker
	worker, err := client.NewWorker("scheduled-worker", hatchet.WithWorkflows(task))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	go scheduleOneTime(client)

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
