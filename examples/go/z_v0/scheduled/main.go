package main

import (
	"context"
	"fmt"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// > Create
// ... normal workflow definition
type printOutput struct{}

func print(ctx context.Context) (result *printOutput, err error) {
	fmt.Println("called print:print")

	return &printOutput{}, nil
}

// ,
func main() {
	// ... initialize client, worker and workflow
	err := godotenv.Load()

	if err != nil {
		panic(err)
	}

	c, err := client.New()

	if err != nil {
		panic(err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)

	if err != nil {
		panic(err)
	}

	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.NoTrigger(),
			Name:        "schedule-workflow",
			Description: "Demonstrates a simple scheduled workflow",
			Steps: []*worker.WorkflowStep{
				worker.Fn(print),
			},
		},
	)

	if err != nil {
		panic(err)
	}

	interrupt := cmdutils.InterruptChan()

	cleanup, err := w.Start()

	if err != nil {
		panic(err)
	}

	// ,

	go func() {
		// 👀 define the scheduled workflow to run in a minute
		schedule, err := c.Schedule().Create(
			context.Background(),
			"schedule-workflow",
			&client.ScheduleOpts{
				// 👀 define the time to run the scheduled workflow, in UTC
				TriggerAt: time.Now().UTC().Add(time.Minute),
				Input: map[string]interface{}{
					"message": "Hello, world!",
				},
				AdditionalMetadata: map[string]string{},
			},
		)

		if err != nil {
			panic(err)
		}

		fmt.Println(schedule.TriggerAt, schedule.WorkflowName)
	}()

	// ... wait for interrupt signal

	<-interrupt

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}

	// ,
}

// !!

func ListScheduledWorkflows() {
	c, err := client.New()

	if err != nil {
		panic(err)
	}

	// > List
	schedules, err := c.Schedule().List(context.Background())
	// !!

	if err != nil {
		panic(err)
	}

	for _, schedule := range *schedules.Rows {
		fmt.Println(schedule.TriggerAt, schedule.WorkflowName)
	}
}

func DeleteScheduledWorkflow(id string) {
	c, err := client.New()

	if err != nil {
		panic(err)
	}

	// > Delete
	// 👀 id is the schedule's metadata id, can get it via schedule.Metadata.Id
	err = c.Schedule().Delete(context.Background(), id)
	// !!

	if err != nil {
		panic(err)
	}
}
