package main

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type streamEventInput struct {
	Index int `json:"index"`
}

type stepOneOutput struct {
	Message string `json:"message"`
}

func StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
	input := &streamEventInput{}

	err = ctx.WorkflowInput(input)

	if err != nil {
		return nil, err
	}

	ctx.StreamEvent([]byte(fmt.Sprintf("This is a stream event %d", input.Index)))

	return &stepOneOutput{
		Message: fmt.Sprintf("This ran at %s", time.Now().String()),
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
			Name:        "stream-event-workflow",
			Description: "This sends a stream event.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(StepOne).SetName("step-one"),
			},
		},
	)

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	_, err = w.Start()

	if err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}

	workflow, err := c.Admin().RunWorkflow("stream-event-workflow", &streamEventInput{
		Index: 0,
	})

	if err != nil {
		panic(err)
	}

	err = c.Subscribe().Stream(interruptCtx, workflow.WorkflowRunId(), func(event client.StreamEvent) error {
		fmt.Println(string(event.Message))

		return nil
	})

	if err != nil {
		panic(err)
	}
}
