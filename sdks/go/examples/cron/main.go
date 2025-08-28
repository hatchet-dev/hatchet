package main

import (
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type CronInput struct {
	Timestamp string `json:"timestamp"`
}

type CronOutput struct {
	JobName    string `json:"job_name"`
	ExecutedAt string `json:"executed_at"`
	NextRun    string `json:"next_run"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Daily cleanup job - runs at 2 AM every day
	dailyCleanup := client.NewStandaloneTask("cleanup-temp-files", func(ctx hatchet.Context, input CronInput) (CronOutput, error) {
		log.Printf("Running daily cleanup at %s", input.Timestamp)

		// Simulate cleanup work
		time.Sleep(2 * time.Second)

		return CronOutput{
			JobName:    "daily-cleanup",
			ExecutedAt: time.Now().Format(time.RFC3339),
			NextRun:    "Next run: tomorrow at 2 AM",
		}, nil
	}, hatchet.WithCron("0 2 * * *")) // 2 AM daily

	// Hourly health check - runs every hour
	healthCheck := client.NewStandaloneTask("check-system-health", func(ctx hatchet.Context, input CronInput) (CronOutput, error) {
		log.Printf("Running health check at %s", input.Timestamp)

		// Simulate health check work
		time.Sleep(500 * time.Millisecond)

		return CronOutput{
			JobName:    "health-check",
			ExecutedAt: time.Now().Format(time.RFC3339),
			NextRun:    "Next run: top of next hour",
		}, nil
	}, hatchet.WithCron("0 * * * *")) // Every hour

	// Weekly report - runs every Monday at 9 AM
	weeklyReport := client.NewStandaloneTask("generate-report", func(ctx hatchet.Context, input CronInput) (CronOutput, error) {
		log.Printf("Generating weekly report at %s", input.Timestamp)

		// Simulate report generation
		time.Sleep(5 * time.Second)

		return CronOutput{
			JobName:    "weekly-report",
			ExecutedAt: time.Now().Format(time.RFC3339),
			NextRun:    "Next run: next Monday at 9 AM",
		}, nil
	}, hatchet.WithCron("0 9 * * 1")) // 9 AM every Monday

	// Multiple cron expressions for business hours monitoring
	businessHoursMonitor := client.NewStandaloneTask("monitor-business-systems", func(ctx hatchet.Context, input CronInput) (CronOutput, error) {
		log.Printf("Monitoring business systems at %s", input.Timestamp)

		return CronOutput{
			JobName:    "business-hours-monitor",
			ExecutedAt: time.Now().Format(time.RFC3339),
			NextRun:    "Next run: next business hour",
		}, nil
	}, hatchet.WithCron(
		"0 9-17 * * 1-5", // Every hour from 9 AM to 5 PM, Monday to Friday
		"0 12 * * 6",     // Saturday at noon
	))

	worker, err := client.NewWorker("cron-worker",
		hatchet.WithWorkflows(dailyCleanup, healthCheck, weeklyReport, businessHoursMonitor),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	log.Println("Starting cron worker...")
	log.Println("Scheduled jobs:")
	log.Println("  - daily-cleanup: 0 2 * * * (2 AM daily)")
	log.Println("  - health-check: 0 * * * * (every hour)")
	log.Println("  - weekly-report: 0 9 * * 1 (9 AM every Monday)")
	log.Println("  - business-hours-monitor: 0 9-17 * * 1-5, 0 12 * * 6 (business hours)")

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
