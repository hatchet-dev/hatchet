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

type userCreateEvent struct {
	Username string            `json:"username"`
	UserId   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type stepOneOutput struct {
	Message string `json:"message"`
}

func StepOne(ctx context.Context, input *userCreateEvent) (result *stepOneOutput, err error) {
	fmt.Println("this ran at: ", time.Now())

	return &stepOneOutput{
		Message: "This ran at: " + time.Now().Format(time.RubyDate),
	}, nil
}

func main() {
	err := godotenv.Load()

	if err != nil {
		panic(err)
	}

	client, err := client.New()

	if err != nil {
		panic(err)
	}

	// Create a worker. This automatically reads in a TemporalClient from .env and workflow files from the .hatchet
	// directory, but this can be customized with the `worker.WithTemporalClient` and `worker.WithWorkflowFiles` options.
	w, err := worker.NewWorker(
		worker.WithClient(
			client,
		),
	)

	if err != nil {
		panic(err)
	}

	err = w.On(
		worker.At(time.Now().Add(time.Second*10)),
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

		at := []time.Time{}

		for i := 0; i < 9; i++ {
			at = append(at, time.Now().Add(time.Second*60+time.Millisecond*10*time.Duration(i)))
		}

		err = client.Admin().ScheduleWorkflow(
			"scheduled-workflow",
			at...,
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
