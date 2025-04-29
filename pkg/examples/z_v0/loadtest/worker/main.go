package main

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type Event struct {
	ID        uint64    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type stepOneOutput struct {
	Message string `json:"message"`
}

func StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
	input := &Event{}

	err = ctx.WorkflowInput(input)

	if err != nil {
		return nil, err
	}

	fmt.Println(input.ID, "delay", time.Since(input.CreatedAt))

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

	w, err := worker.NewWorker(
		worker.WithClient(
			client,
		),
	)

	if err != nil {
		panic(err)
	}

	err = w.On(
		worker.Event("test:event"),
		&worker.WorkflowJob{
			Name:        "scheduled-workflow",
			Description: "This runs at a scheduled time.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(StepOne).SetName("step-one"),
			},
		},
	)

	if err != nil {
		panic(err)
	}

	ch := cmdutils.InterruptChan()

	cleanup, err := w.Start()
	if err != nil {
		panic(fmt.Errorf("error starting worker: %w", err))
	}

	<-ch

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}
}
