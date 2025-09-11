package main

import (
	"fmt"

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
		panic(fmt.Sprintf("error creating client: %v", err))
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
		worker.WithMaxRuns(1),
	)
	if err != nil {
		panic(fmt.Sprintf("error creating worker: %v", err))
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
		panic(fmt.Sprintf("error registering workflow: %v", err))
	}

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	cleanup, err := w.Start()
	if err != nil {
		panic(fmt.Sprintf("error starting worker: %v", err))
	}

	<-interruptCtx.Done()
	if err := cleanup(); err != nil {
		panic(err)
	}
}
