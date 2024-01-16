package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserID   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type stepOutput struct {
	Message string `json:"message"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	events := make(chan string, 50)
	if err := run(cmdutils.InterruptChan(), events); err != nil {
		panic(err)
	}
}

func run(ch <-chan interface{}, events chan<- string) error {
	c, err := client.New()

	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	// Create a worker. This automatically reads in a TemporalClient from .env and workflow files from the .hatchet
	// directory, but this can be customized with the `worker.WithTemporalClient` and `worker.WithWorkflowFiles` options.
	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)
	if err != nil {
		return fmt.Errorf("error creating worker: %w", err)
	}

	testSvc := w.NewService("test")

	err = testSvc.On(
		worker.Events("user:create:simple"),
		&worker.WorkflowJob{
			Name:        "post-user-update",
			Description: "This runs after an update to the user model.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {
					input := &userCreateEvent{}
					ctx.WorkflowInput(input)

					return &stepOutput{
						Message: "Step 1 got username: " + input.Username,
					}, nil
				},
				).SetName("step-one"),
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {
					input := &userCreateEvent{}
					ctx.WorkflowInput(input)

					return &stepOutput{
						Message: "Step 2 got username: " + input.Username,
					}, nil
				}).SetName("step-two"),
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {
					step1Out := &stepOutput{}
					ctx.StepOutput("step-one", step1Out)

					step2Out := &stepOutput{}
					ctx.StepOutput("step-two", step2Out)

					return &stepOutput{
						Message: "Step 3: has parents 1 and 2:" + step1Out.Message + ", " + step2Out.Message,
					}, nil
				}).SetName("step-three").AddParents("step-one", "step-two"),
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {
					step1Out := &stepOutput{}
					ctx.StepOutput("step-one", step1Out)

					step3Out := &stepOutput{}
					ctx.StepOutput("step-three", step3Out)

					return &stepOutput{
						Message: "Step 4: has parents 1 and 3" + step1Out.Message + ", " + step3Out.Message,
					}, nil
				}).SetName("step-four").AddParents("step-one", "step-three"),
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {
					step4Out := &stepOutput{}
					ctx.StepOutput("step-four", step4Out)

					return &stepOutput{
						Message: "Step 5: has parent 4" + step4Out.Message,
					}, nil
				}).SetName("step-five").AddParents("step-four"),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error registering workflow: %w", err)
	}

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(ch)
	defer cancel()

	go func() {
		err = w.Start(interruptCtx)

		if err != nil {
			panic(err)
		}

		cancel()
	}()

	testEvent := userCreateEvent{
		Username: "echo-test",
		UserID:   "1234",
		Data: map[string]string{
			"test": "test",
		},
	}

	log.Printf("pushing event user:create:simple")

	// push an event
	err = c.Event().Push(
		context.Background(),
		"user:create:simple",
		testEvent,
	)

	if err != nil {
		return fmt.Errorf("error pushing event: %w", err)
	}

	for {
		select {
		case <-interruptCtx.Done():
			return nil
		default:
			time.Sleep(time.Second)
		}
	}
}
