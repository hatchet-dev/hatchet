package main

import (
	"context"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
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

	// > Workflow definition cron trigger
	dailyCleanup := client.NewStandaloneTask("cleanup-temp-files", func(ctx hatchet.Context, input CronInput) (CronOutput, error) {
		log.Printf("Running daily cleanup at %s", input.Timestamp)

		time.Sleep(2 * time.Second)

		return CronOutput{
			JobName:    "daily-cleanup",
			ExecutedAt: time.Now().Format(time.RFC3339),
			NextRun:    "Next run: tomorrow at 2 AM",
		}, nil
	},
		hatchet.WithWorkflowCron("0 2 * * *"),
		hatchet.WithWorkflowCronInput(CronInput{
			Timestamp: time.Now().Format(time.RFC3339),
		}),
		hatchet.WithWorkflowDescription("Daily cleanup and maintenance tasks"),
	)

	healthCheck := client.NewStandaloneTask("check-system-health", func(ctx hatchet.Context, input CronInput) (CronOutput, error) {
		log.Printf("Running health check at %s", input.Timestamp)

		time.Sleep(500 * time.Millisecond)

		return CronOutput{
			JobName:    "health-check",
			ExecutedAt: time.Now().Format(time.RFC3339),
			NextRun:    "Next run: top of next hour",
		}, nil
	},
		hatchet.WithWorkflowCron("0 * * * *"),
		hatchet.WithWorkflowDescription("Hourly system health monitoring"),
	)

	weeklyReport := client.NewStandaloneTask("generate-report", func(ctx hatchet.Context, input CronInput) (CronOutput, error) {
		log.Printf("Generating weekly report at %s", input.Timestamp)

		time.Sleep(5 * time.Second)

		return CronOutput{
			JobName:    "weekly-report",
			ExecutedAt: time.Now().Format(time.RFC3339),
			NextRun:    "Next run: next Monday at 9 AM",
		}, nil
	},
		hatchet.WithWorkflowCron("0 9 * * 1"),
		hatchet.WithWorkflowDescription("Weekly business metrics report"),
	)

	businessHoursMonitor := client.NewStandaloneTask("monitor-business-systems", func(ctx hatchet.Context, input CronInput) (CronOutput, error) {
		log.Printf("Monitoring business systems at %s", input.Timestamp)

		return CronOutput{
			JobName:    "business-hours-monitor",
			ExecutedAt: time.Now().Format(time.RFC3339),
			NextRun:    "Next run: next business hour",
		}, nil
	},
		hatchet.WithWorkflowCron(
			"0 9-17 * * 1-5",
			"0 12 * * 6",
		),
		hatchet.WithWorkflowDescription("Monitor systems during business hours"),
	)

	worker, err := client.NewWorker("cron-worker",
		hatchet.WithWorkflows(dailyCleanup, healthCheck, weeklyReport, businessHoursMonitor),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	_ = func() error {
		// > Create
		createdCron, err := client.Crons().Create(context.Background(), "cleanup-temp-files", features.CreateCronTrigger{
			Name:       "daily-cleanup",
			Expression: "0 0 * * *",
			Input: map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
			},
			AdditionalMetadata: map[string]interface{}{
				"description": "Daily cleanup and maintenance tasks",
			},
		})
		if err != nil {
			return err
		}

		// > Delete
		err = client.Crons().Delete(context.Background(), createdCron.Metadata.Id)
		if err != nil {
			return err
		}

		// > List
		cronList, err := client.Crons().List(context.Background(), rest.CronWorkflowListParams{
			AdditionalMetadata: &[]string{"description:Daily cleanup and maintenance tasks"},
		})
		if err != nil {
			return err
		}

		_ = cronList

		return nil
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
