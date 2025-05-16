package main

import (
	"context"
	"fmt"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type concurrencyLimitEvent struct {
	Index int `json:"index"`
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
		worker.Events("concurrency-test-event"),
		&worker.WorkflowJob{
			Name:        "concurrency-limit",
			Description: "This limits concurrency to 1 run at a time.",
			Concurrency: worker.Concurrency(getConcurrencyKey).MaxRuns(1),
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					<-ctx.Done()
					fmt.Println("context done, returning")
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
			if err := cleanup(); err != nil {
				panic(fmt.Errorf("error cleaning up: %w", err))
			}
			return
		case <-time.After(2 * time.Second): // timeout
		}

		firstEvent := concurrencyLimitEvent{
			Index: 0,
		}

		// push an event
		err = c.Event().Push(
			context.Background(),
			"concurrency-test-event",
			firstEvent,
			nil,
			nil,
		)

		if err != nil {
			panic(err)
		}

		select {
		case <-interruptCtx.Done(): // context cancelled
			fmt.Println("interrupted")
			return
		case <-time.After(10 * time.Second): // timeout
		}

		// push a second event
		err = c.Event().Push(
			context.Background(),
			"concurrency-test-event",
			concurrencyLimitEvent{
				Index: 1,
			},
			nil,
			nil,
		)

		if err != nil {
			panic(err)
		}
	}()

	for {
		select {
		case <-interruptCtx.Done():
			return nil
		default:
			time.Sleep(time.Second)
		}
	}
}
