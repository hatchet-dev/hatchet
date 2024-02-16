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

type stepOneOutput struct {
	Message string `json:"message"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	events := make(chan string, 50)
	ctx, _ := cmdutils.NewInterruptContext()
	if err := run(ctx, events); err != nil {
		panic(err)
	}
}

func run(interruptCtx context.Context, events chan<- string) error {
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
		return fmt.Errorf("error registering workflow: %w", err)
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
		err = c.Event().Push(
			context.Background(),
			"user:create:middleware",
			testEvent,
		)
		if err != nil {
			panic(fmt.Errorf("error pushing event: %w", err))
		}
	}()

	err = w.Start(interruptCtx)

	if err != nil {
		panic(err)
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
