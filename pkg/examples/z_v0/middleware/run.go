package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

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

	w.Use(func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {
		log.Printf("1st-middleware")
		events <- "1st-middleware"
		ctx.SetContext(context.WithValue(ctx.GetContext(), "testkey", "testvalue"))
		return next(ctx)
	})

	w.Use(func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {
		log.Printf("2nd-middleware")
		events <- "2nd-middleware"

		// time the function duration
		start := time.Now()
		err := next(ctx)
		duration := time.Since(start)
		fmt.Printf("step function took %s\n", duration)
		return err
	})

	testSvc := w.NewService("test")

	testSvc.Use(func(ctx worker.HatchetContext, next func(worker.HatchetContext) error) error {
		events <- "svc-middleware"
		ctx.SetContext(context.WithValue(ctx.GetContext(), "svckey", "svcvalue"))
		return next(ctx)
	})

	err = testSvc.On(
		worker.Events("user:create:middleware"),
		&worker.WorkflowJob{
			Name:        "middleware",
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

					testVal := ctx.Value("testkey").(string)
					events <- testVal
					svcVal := ctx.Value("svckey").(string)
					events <- svcVal

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
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}

	go func() {
		log.Printf("pushing event user:create:middleware")

		testEvent := userCreateEvent{
			Username: "echo-test",
			UserID:   "1234",
			Data: map[string]string{
				"test": "test",
			},
		}

		// push an event
		err := c.Event().Push(
			context.Background(),
			"user:create:middleware",
			testEvent,
			nil,
			nil,
		)
		if err != nil {
			panic(fmt.Errorf("error pushing event: %w", err))
		}
	}()

	cleanup, err := w.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting worker: %w", err)
	}

	return cleanup, nil
}
