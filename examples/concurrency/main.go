package main

import (
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
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
	interrupt := cmdutils.InterruptChan()

	cleanup, err := run(events)
	if err != nil {
		panic(err)
	}

	<-interrupt

	if err := cleanup(); err != nil {

		panic(fmt.Errorf("error cleaning up: %w", err))
	}
}

func run(events chan<- string) (func() error, error) {
	c, err := client.New()

	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating worker: %w", err)
	}

	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.Events("user:create:concurrency"),
			Name:        "simple-concurrency",
			Description: "This runs to test concurrency.",
			Concurrency: worker.Concurrency(getConcurrencyKey).MaxRuns(1).LimitStrategy(types.CancelInProgress),
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					input := &userCreateEvent{}

					err = ctx.WorkflowInput(input)

					// we sleep to simulate a long running task
					time.Sleep(10 * time.Second)

					if err != nil {

						return nil, err
					}

					if ctx.Err() != nil {
						return nil, ctx.Err()
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

					if ctx.Err() != nil {
						return nil, ctx.Err()
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
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}
	testEvent := userCreateEvent{
		Username: "echo-test",
		UserID:   "1234",
		Data: map[string]string{
			"test": "test",
		},
	}
	go func() {
		// do this 10 times to test concurrency
		for i := 0; i < 10; i++ {

			wfr_id, err := c.Admin().RunWorkflow("simple-concurrency", testEvent)

			log.Println("Starting workflow run id: ", wfr_id)

			if err != nil {
				panic(fmt.Errorf("error running workflow: %w", err))
			}

		}
	}()

	cleanup, err := w.Start()
	if err != nil {
		panic(err)
	}

	return cleanup, nil
}

func getConcurrencyKey(ctx worker.HatchetContext) (string, error) {
	return "concurrency", nil
}
