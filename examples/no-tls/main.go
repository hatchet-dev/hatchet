package main

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type stepOutput struct{}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	c, err := client.New()

	if err != nil {
		panic(fmt.Errorf("error creating client: %w", err))
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
		worker.WithMaxRuns(1),
	)
	if err != nil {
		panic(fmt.Errorf("error creating worker: %w", err))
	}

	testSvc := w.NewService("test")

	err = testSvc.On(
		worker.Events("simple"),
		&worker.WorkflowJob{
			Name:        "simple-workflow",
			Description: "Simple one-step workflow.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {
					fmt.Println("executed step 1")

					return &stepOutput{}, nil
				},
				).SetName("step-one"),
			},
		},
	)
	if err != nil {
		panic(fmt.Errorf("error registering workflow: %w", err))
	}

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	cleanup, err := w.Start()
	if err != nil {
		panic(fmt.Errorf("error starting worker: %w", err))
	}

	for {
		select {
		case <-interruptCtx.Done():
			cleanup() // nolint: errcheck
		default:
			time.Sleep(time.Second)
		}
	}
}
