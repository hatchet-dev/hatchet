package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/joho/godotenv"
)

type scheduledInput struct {
	ScheduledAt time.Time `json:"scheduled_at"`
	ExecuteAt   time.Time `json:"scheduled_for"`
}

type stepOneOutput struct {
	Message string `json:"message"`
}

func StepOne(ctx context.Context, input *scheduledInput) (result *stepOneOutput, err error) {
	// get time between execute at and scheduled at
	timeBetween := time.Since(input.ScheduledAt)

	return &stepOneOutput{
		Message: fmt.Sprintf("This ran %s after scheduling", timeBetween),
	}, nil
}

func main() {
	err := godotenv.Load()

	if err != nil {
		panic(err)
	}

	c, err := client.New()

	if err != nil {
		panic(err)
	}

	// Create a worker. This automatically reads in a TemporalClient from .env and workflow files from the .hatchet
	// directory, but this can be customized with the `worker.WithTemporalClient` and `worker.WithWorkflowFiles` options.
	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)

	if err != nil {
		panic(err)
	}

	err = w.On(
		worker.NoTrigger(),
		&worker.WorkflowJob{
			Name:        "scheduled-workflow",
			Description: "This runs at a scheduled time.",
			Steps: []worker.WorkflowStep{
				worker.Fn(StepOne).SetName("step-one"),
			},
		},
	)

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	go func() {
		err = w.Start(interruptCtx)

		if err != nil {
			panic(err)
		}

		cancel()
	}()

	go func() {
		time.Sleep(5 * time.Second)

		executeAt := time.Now().Add(time.Second * 10)

		err = c.Admin().ScheduleWorkflow(
			"scheduled-workflow",
			client.WithSchedules(executeAt),
			client.WithInput(&scheduledInput{
				ScheduledAt: time.Now(),
				ExecuteAt:   executeAt,
			}),
		)

		if err != nil {
			panic(err)
		}
	}()

	for {
		select {
		case <-interruptCtx.Done():
			return
		default:
			time.Sleep(time.Second)
		}
	}
}
