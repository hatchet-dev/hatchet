package rampup

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

func runWorker(ctx context.Context, client client.Client, concurrency int, maxAcceptableDuration time.Duration, workerStarted chan<- time.Time, errChan chan<- error, resultChan chan<- Event) (int64, int64) {

	fmt.Println("running")

	w, err := worker.NewWorker(
		worker.WithClient(
			client,
		),
		worker.WithLogLevel("warn"),
		worker.WithMaxRuns(200),
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
	fmt.Println("defining worker")
	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.Event("load-test:event"),
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

					l.Debug().Msgf("executing %d took %s", input.ID, took)

					if took > maxAcceptableDuration {
						errChan <- fmt.Errorf("event %d took too long to execute: %s", input.ID, took)
					}

					mx.Lock()
					defer mx.Unlock()

					// detect duplicate in executed slice
					var duplicate bool
					for i := 0; i < len(executed)-1; i++ {
						if executed[i] == input.ID {
							duplicate = true
						}
					}
					if duplicate {
						l.Error().Str("step-run-id", ctx.StepRunId()).Msgf("duplicate %d", input.ID)
						e := fmt.Errorf("duplicate %d", input.ID)
						errChan <- e
						return nil, e

					}

					uniques++
					resultChan <- input

					count++
					executed = append(executed, input.ID)

					return &stepOneOutput{
						Message: "This ran at: " + time.Now().Format(time.RFC3339Nano),
					}, nil
				}).SetName("step-one"),
			},
		},
	)

	fmt.Println("registered workflow")

	if err != nil {
		panic(err)
	}
	fmt.Println("starting worker")
	cleanup, err := w.Start()
	if err != nil {
		panic(fmt.Errorf("error starting worker: %w", err))
	}
	fmt.Println("worker started")
	workerStarted <- time.Now()
	fmt.Println("waiting for context to be done")
	<-ctx.Done()

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}

	mx.Lock()
	defer mx.Unlock()
	return count, uniques
}
