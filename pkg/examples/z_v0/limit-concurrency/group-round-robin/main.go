package main

import (
	"context"
	"fmt"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type concurrencyLimitEvent struct {
	UserId int `json:"user_id"`
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
	input := &concurrencyLimitEvent{}
	err := ctx.WorkflowInput(input)

	if err != nil {
		return "", fmt.Errorf("error getting input: %w", err)
	}

	return fmt.Sprintf("%d", input.UserId), nil
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
	)
	if err != nil {
		return fmt.Errorf("error creating worker: %w", err)
	}

	testSvc := w.NewService("test")

	err = testSvc.On(
		worker.Events("concurrency-test-event-rr"),
		&worker.WorkflowJob{
			Name:        "concurrency-limit-round-robin",
			Description: "This limits concurrency to 2 runs at a time.",
			Concurrency: worker.Concurrency(getConcurrencyKey).MaxRuns(2).LimitStrategy(types.GroupRoundRobin),
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					input := &concurrencyLimitEvent{}

					err = ctx.WorkflowInput(input)

					if err != nil {
						return nil, fmt.Errorf("error getting input: %w", err)
					}

					fmt.Println("received event", input.UserId)

					time.Sleep(5 * time.Second)

					fmt.Println("processed event", input.UserId)

					return nil, nil
				},
				).SetName("step-one"),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error registering workflow: %w", err)
	}

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(ch)
	defer cancel()

	cleanup, err := w.Start()
	if err != nil {
		return fmt.Errorf("error starting worker: %w", err)
	}

	go func() {
		// sleep with interrupt context
		select {
		case <-interruptCtx.Done(): // context cancelled
			fmt.Println("interrupted")
			return
		case <-time.After(2 * time.Second): // timeout
		}

		for i := 0; i < 20; i++ {
			var event concurrencyLimitEvent

			if i < 10 {
				event = concurrencyLimitEvent{0}
			} else {
				event = concurrencyLimitEvent{1}
			}

			c.Event().Push(
				context.Background(),
				"concurrency-test-event-rr",
				event,
				nil,
				nil,
			)
		}

		select {
		case <-interruptCtx.Done(): // context cancelled
			fmt.Println("interrupted")
			return
		case <-time.After(10 * time.Second): //timeout
		}
	}()

	for {
		select {
		case <-interruptCtx.Done():
			if err := cleanup(); err != nil {
				return fmt.Errorf("error cleaning up: %w", err)
			}
			return nil
		default:
			time.Sleep(time.Second)
		}
	}
}
