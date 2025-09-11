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

	ch := cmdutils.InterruptChan()

	if err != nil {
		panic(err)
	}
	workflowName := "simple-bulk-workflow"
	c, err := client.New()

	if err != nil {
		panic(fmt.Errorf("error creating client: %w", err))
	}

	_, err = registerWorkflow(c, workflowName)

	if err != nil {
		panic(fmt.Errorf("error registering workflow: %w", err))
	}

	quantity := 999

	overallStart := time.Now()
	iterations := 10
	for i := 0; i < iterations; i++ {
		startTime := time.Now()

		fmt.Printf("Running the %dth bulk workflow \n", i)

		err = runBulk(workflowName, quantity)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Time taken to queue %dth bulk workflow: %v\n", i, time.Since(startTime))
	}
	fmt.Println("Overall time taken: ", time.Since(overallStart))
	fmt.Printf("That is %d workflows per second\n", int(float64(quantity*iterations)/time.Since(overallStart).Seconds()))
	fmt.Println("Starting the worker")

	// err = runSingles(workflowName, quantity)
	// if err != nil {
	// 	panic(err)
	// }

	if err != nil {
		panic(fmt.Errorf("error creating client: %w", err))
	}

	// I want to start the wofklow worker here

	w, err := registerWorkflow(c, workflowName)
	if err != nil {
		panic(fmt.Errorf("error creating worker: %w", err))
	}

	cleanup, err := w.Start()
	fmt.Println("Starting the worker")

	if err != nil {
		panic(fmt.Errorf("error starting worker: %w", err))
	}

	<-ch

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}

}

func getConcurrencyKey(ctx worker.HatchetContext) (string, error) {
	return "my-key", nil
}

func registerWorkflow(c client.Client, workflowName string) (w *worker.Worker, err error) {

	w, err = worker.NewWorker(
		worker.WithClient(
			c,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating worker: %w", err)
	}

	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.Events("user:create:bulk-simple"),
			Name:        workflowName,
			Concurrency: worker.Concurrency(getConcurrencyKey).MaxRuns(200).LimitStrategy(types.GroupRoundRobin),
			Description: "This runs after an update to the user model.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					input := &userCreateEvent{}

					err = ctx.WorkflowInput(input)

					if err != nil {
						return nil, err
					}

					log.Printf("step-one")

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
	return w, nil
}
