package main

import (
	"context"
	"fmt"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// > Workflow Definition Cron Trigger
// ... normal workflow definition
type printOutput struct{}

func print(ctx context.Context) (result *printOutput, err error) {
	fmt.Println("called print:print")

	return &printOutput{}, nil
}

// ,
func main() {
	// ... initialize client and worker
	err := godotenv.Load()

	if err != nil {
		panic(err)
	}

	client, err := client.New()

	if err != nil {
		panic(err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			client,
		),
	)

	if err != nil {
		panic(err)
	}

	// ,
	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			// 👀 define the cron expression to run every minute
			On:          worker.Cron("* * * * *"),
			Name:        "cron-workflow",
			Description: "Demonstrates a simple cron workflow",
			Steps: []*worker.WorkflowStep{
				worker.Fn(print),
			},
		},
	)

	if err != nil {
		panic(err)
	}

	// ... start worker

	interrupt := cmdutils.InterruptChan()

	cleanup, err := w.Start()

	if err != nil {
		panic(err)
	}

	<-interrupt

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}

	// ,
}

// !!
