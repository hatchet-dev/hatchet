package main

import (
	"context"
	"fmt"
	"log"

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

func getConcurrencyKey(ctx worker.HatchetContext) (string, error) {
	return "user-create", nil
}

type retryWorkflow struct {
	retries int
}

func (r *retryWorkflow) StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
	input := &userCreateEvent{}

	err = ctx.WorkflowInput(input)

	if err != nil {
		return nil, err
	}

	if r.retries < 2 {
		r.retries++
		return nil, fmt.Errorf("error")
	}

	log.Printf("finished step-one")
	return &stepOneOutput{
		Message: "Username is: " + input.Username,
	}, nil
}

func run(ch <-chan interface{}, events chan<- string) error {
	c, err := client.New()

	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
		worker.WithMaxRuns(1),
	)
	if err != nil {
		return fmt.Errorf("error creating worker: %w", err)
	}

	testSvc := w.NewService("test")

	wk := &retryWorkflow{}

	err = testSvc.On(
		worker.Events("user:create:simple"),
		&worker.WorkflowJob{
			Name:        "simple",
			Description: "This runs after an update to the user model.",
			Concurrency: worker.Concurrency(getConcurrencyKey),
			Steps: []*worker.WorkflowStep{
				worker.Fn(wk.StepOne).SetName("step-one").SetRetries(4),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error registering workflow: %w", err)
	}

	cleanup, err := w.Start()
	if err != nil {
		return fmt.Errorf("error starting worker: %w", err)
	}

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

	<-ch

	if err := cleanup(); err != nil {
		return fmt.Errorf("error cleaning up worker: %w", err)
	}

	return nil
}
