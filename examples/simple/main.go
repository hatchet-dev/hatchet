package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/joho/godotenv"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserID   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type stepOneOutput struct {
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
			Name:        "simple",
			Description: "This runs after an update to the user model.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					input := &userCreateEvent{}

					err = ctx.WorkflowInput(input)

					if err != nil {
						return nil, err
					}

					log.Printf("step-one")
					events <- "step-one"

					return &stepOneOutput{
						Message: "Username is: " + input.Username,
					}, nil
				},
				).SetName("step-one"),
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					input := &stepOneOutput{}
					err = ctx.StepOutput("step-one", input)

					if err != nil {
						return nil, err
					}

					log.Printf("step-two")
					events <- "step-two"

					return &stepOneOutput{
						Message: "Above message is: " + input.Message,
					}, nil
				}).SetName("step-two").AddParents("step-one"),
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
