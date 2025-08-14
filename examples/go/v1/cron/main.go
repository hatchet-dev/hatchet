package main

import (
	"log"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type CronInput struct {
	Timestamp string `json:"timestamp"`
}

type CronOutput struct {
	JobName     string `json:"job_name"`
	ExecutedAt  string `json:"executed_at"`
	NextRun     string `json:"next_run"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Daily cleanup job - runs at 2 AM every day
	dailyCleanup := client.NewWorkflow("daily-cleanup", 
		hatchet.WithWorkflowCron("0 2 * * *"), // 2 AM daily
		hatchet.WithWorkflowDescription("Daily cleanup and maintenance tasks"),
	)

	dailyCleanup.NewTask("cleanup-temp-files", func(ctx hatchet.Context, input CronInput) (CronOutput, error) {
		log.Printf("Running daily cleanup at %s", input.Timestamp)
		
		// Simulate cleanup work
		time.Sleep(2 * time.Second)
		
		return CronOutput{
			JobName:    "daily-cleanup",
			ExecutedAt: time.Now().Format(time.RFC3339),
			NextRun:    "Next run: tomorrow at 2 AM",
		}, nil
	})

	// Hourly health check - runs every hour
	healthCheck := client.NewWorkflow("health-check",
		hatchet.WithWorkflowCron("0 * * * *"), // Every hour
		hatchet.WithWorkflowDescription("Hourly system health monitoring"),
	)

	healthCheck.NewTask("check-system-health", func(ctx hatchet.Context, input CronInput) (CronOutput, error) {
		log.Printf("Running health check at %s", input.Timestamp)
		
		// Simulate health check work
		time.Sleep(500 * time.Millisecond)
		
		return CronOutput{
			JobName:    "health-check",
			ExecutedAt: time.Now().Format(time.RFC3339),
			NextRun:    "Next run: top of next hour",
		}, nil
	})

	// Weekly report - runs every Monday at 9 AM
	weeklyReport := client.NewWorkflow("weekly-report",
		hatchet.WithWorkflowCron("0 9 * * 1"), // 9 AM every Monday
		hatchet.WithWorkflowDescription("Weekly business metrics report"),
	)

	weeklyReport.NewTask("generate-report", func(ctx hatchet.Context, input CronInput) (CronOutput, error) {
		log.Printf("Generating weekly report at %s", input.Timestamp)
		
		// Simulate report generation
		time.Sleep(5 * time.Second)
		
		return CronOutput{
			JobName:    "weekly-report",
			ExecutedAt: time.Now().Format(time.RFC3339),
			NextRun:    "Next run: next Monday at 9 AM",
		}, nil
	})

	// Multiple cron expressions for business hours monitoring
	businessHoursMonitor := client.NewWorkflow("business-hours-monitor",
		hatchet.WithWorkflowCron(
			"0 9-17 * * 1-5",   // Every hour from 9 AM to 5 PM, Monday to Friday
			"0 12 * * 6",       // Saturday at noon
		),
		hatchet.WithWorkflowDescription("Monitor systems during business hours"),
	)

	businessHoursMonitor.NewTask("monitor-business-systems", func(ctx hatchet.Context, input CronInput) (CronOutput, error) {
		log.Printf("Monitoring business systems at %s", input.Timestamp)
		
		return CronOutput{
			JobName:    "business-hours-monitor",
			ExecutedAt: time.Now().Format(time.RFC3339),
			NextRun:    "Next run: next business hour",
		}, nil
	})

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

	if err := worker.StartBlocking(); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}