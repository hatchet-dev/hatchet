package main

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type stepOneOutput struct {
	Message string `json:"message"`
}

func StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
	return nil, fmt.Errorf("test on failure")
}

func OnFailure(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
	return &stepOneOutput{
		Message: "Failure!",
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
			Name:        "on-failure-workflow",
			Description: "This runs at a scheduled time.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(StepOne).SetName("step-one"),
			},
			OnFailure: &worker.WorkflowJob{
				Name:        "scheduled-workflow-failure",
				Description: "This runs when the scheduled workflow fails.",
				Steps: []*worker.WorkflowStep{
					worker.Fn(OnFailure).SetName("on-failure"),
				},
			},
		},
	)

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	cleanup, err := w.Start()
	if err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}

	for {
		select {
		case <-interruptCtx.Done():
			if err := cleanup(); err != nil {
				panic(fmt.Errorf("error cleaning up: %w", err))
			}
			return
		default:
			time.Sleep(time.Second)
		}
	}
}
