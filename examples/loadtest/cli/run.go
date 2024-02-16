package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type stepOneOutput struct {
	Message string `json:"message"`
}

func getConcurrencyKey(ctx worker.HatchetContext) (string, error) {
	return "my-key", nil
}

func run(ctx context.Context, delay time.Duration, executions chan<- time.Duration, concurrency int) (int64, int64) {
	c, err := client.New()

	if err != nil {
		panic(err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)

	if err != nil {
		panic(err)
	}

	mx := sync.Mutex{}
	var count int64
	var uniques int64
	var executed []int64

	var concurrencyOpts *worker.WorkflowConcurrency
	if concurrency > 0 {
		concurrencyOpts = worker.Concurrency(getConcurrencyKey).MaxRuns(int32(concurrency))
	}

	err = w.On(
		worker.Event("load-test:event"),
		&worker.WorkflowJob{
			Name:        "load-test",
			Description: "Load testing",
			Concurrency: concurrencyOpts,
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					var input Event
					err = ctx.WorkflowInput(&input)
					if err != nil {
						return nil, err
					}

					took := time.Since(input.CreatedAt)
					fmt.Println("executing", input.ID, "took", took)

					mx.Lock()
					executions <- took
					// detect duplicate in executed slice
					var duplicate bool
					for i := 0; i < len(executed)-1; i++ {
						if executed[i] == input.ID {
							duplicate = true
							fmt.Println("DUPLICATE:", input.ID)
						}
					}
					if !duplicate {
						uniques += 1
					}
					count += 1
					executed = append(executed, input.ID)
					mx.Unlock()

					time.Sleep(delay)

					return &stepOneOutput{
						Message: "This ran at: " + time.Now().Format(time.RFC3339Nano),
					}, nil
				}).SetName("step-one"),
			},
		},
	)

	if err != nil {
		panic(err)
	}

	go func() {
		err = w.Start(ctx)

		if err != nil {
			panic(err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			mx.Lock()
			defer mx.Unlock()
			return count, uniques
		default:
			time.Sleep(time.Second)
		}
	}
}
