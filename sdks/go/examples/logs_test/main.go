package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type LogsTestInput struct {
	Message string `json:"message"`
}

type LogsTestOutput struct {
	Result string `json:"result"`
}

func main() {
	// > Create a new Hatchet client
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}
	// !!

	var since_log_time time.Time

	// > Create a new standalone task with logging during execution
	task := client.NewStandaloneTask("logs-test", func(ctx hatchet.Context, input LogsTestInput) (LogsTestOutput, error) {
		ctx.Log("Starting task execution")
		time.Sleep(5 * time.Second)

		since_log_time = time.Now()

		ctx.Log("Logging input: " + input.Message)
		ctx.Log("Task completed successfully")
		return LogsTestOutput{
			Result: "Task completed successfully",
		}, nil
	})
	// !!

	// > Create a new worker
	worker, err := client.NewWorker("logs-test-worker",
		hatchet.WithWorkflows(task),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}
	// !!

	runTaskAndFetchLogs := func() error {
		// > Run the task
		result, err := task.Run(context.Background(), LogsTestInput{Message: "Invalid input received!"})
		if err != nil {
			return err
		}
		// !!

		time.Sleep(2 * time.Second)

		// > Get the task run details
		runDetails, err := client.Runs().Get(context.Background(), result.RunId)
		if err != nil {
			return err
		}
		// !!

		if len(runDetails.Tasks) > 0 {
			taskRunId := runDetails.Tasks[0].TaskExternalId

			fmt.Printf("\nTask Run ID: %s\n", taskRunId)

			// > Fetch logs for the task run, with a since filter
			logs, err := client.Logs().List(context.Background(), taskRunId, &rest.V1LogLineListParams{
				Since: &since_log_time,
			})
			if err != nil {
				return err
			}
			// !!

			if logs != nil && logs.Rows != nil {
				fmt.Println("\nTask Logs:")
				for _, logLine := range *logs.Rows {
					fmt.Printf("[%s] %s\n", logLine.CreatedAt.Format(time.RFC3339), logLine.Message)
				}
			} else {
				fmt.Println("No logs found")
			}
		}

		return nil
	}

	// > Start the worker in a goroutine
	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	go func() {
		if err := worker.StartBlocking(interruptCtx); err != nil {
			log.Printf("worker error: %v", err)
		}
	}()
	// !!

	if err := runTaskAndFetchLogs(); err != nil {
		log.Printf("error: %v", err)
	}

	<-interruptCtx.Done()
}
