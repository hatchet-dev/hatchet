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
	wfrIds := make(chan *client.Workflow, 50)
	interrupt := cmdutils.InterruptChan()
	c, err := client.New()

	if err != nil {
		log.Fatalf("error creating client: %v", err)
	}
	cleanup, err := run(c, events, wfrIds)
	if err != nil {
		panic(err)
	}
selectLoop:
	for {
		select {

		case <-interrupt:
			log.Print("Interrupted")
			break selectLoop
		case wfrId := <-wfrIds:
			log.Printf("Workflow run id: %s", wfrId.WorkflowRunId())
			wfResult, err := wfrId.Result()
			if err != nil {

				if err.Error() == "step output for step-one not found" {
					log.Printf("Step output for step-one not found because it was cancelled due to CANCELLED_BY_CONCURRENCY_LIMIT")
					continue
				}
				panic(fmt.Errorf("error getting workflow run result: %w", err))
			}

			stepOneOutput := &stepOneOutput{}

			err = wfResult.StepOutput("step-one", stepOneOutput)

			if err != nil {
				if err.Error() == "step run failed: this step run was cancelled due to CANCELLED_BY_CONCURRENCY_LIMIT" {
					log.Printf("Workflow run was cancelled due to CANCELLED_BY_CONCURRENCY_LIMIT")
					continue
				}
				if err.Error() == "step output for step-one not found" {
					log.Printf("Step output for step-one not found because it was cancelled due to CANCELLED_BY_CONCURRENCY_LIMIT")
					continue
				}
				panic(fmt.Errorf("error getting workflow run result: %w", err))
			}
		case e := <-events:
			log.Printf("Event: %s", e)
		}
	}

	if err := cleanup(); err != nil {

		panic(fmt.Errorf("error cleaning up: %w", err))
	}
}

func run(c client.Client, events chan<- string, wfrIds chan<- *client.Workflow) (func() error, error) {

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
					time.Sleep(5 * time.Second)

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

						if err.Error() == "step run failed: this step run was cancelled due to CANCELLED_BY_CONCURRENCY_LIMIT" {
							return nil, nil
						}

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

			log.Println("Starting workflow run id: ", wfr_id.WorkflowRunId())

			if err != nil {
				panic(fmt.Errorf("error running workflow: %w", err))
			}

			wfrIds <- wfr_id

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
