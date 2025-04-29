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

// > OnFailure Step
// This workflow will fail because the step will throw an error
// we define an onFailure step to handle this case

func StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
	// 👀 this step will always raise an exception
	return nil, fmt.Errorf("test on failure")
}

func OnFailure(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
	// run cleanup code or notifications here

	// 👀 you can access the error from the failed step(s) like this
	fmt.Println(ctx.StepRunErrors())

	return &stepOneOutput{
		Message: "Failure!",
	}, nil
}

func main() {
	// ...
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

	// 👀 we define an onFailure step to handle this case
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

	// ...

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
	// ,
}


