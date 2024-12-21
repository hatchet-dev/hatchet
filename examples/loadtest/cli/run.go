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

func runWorker(ctx context.Context, c client.Client, delay time.Duration, executions chan<- time.Duration, concurrency int, executedChan chan<- int64, duplicateChan chan<- int64) int64 {

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
		worker.WithMaxRuns(200),
	)

	if err != nil {
		panic(err)
	}

	mx := sync.Mutex{}
	var uniques int64
	var executed []int64

	var concurrencyOpts *worker.WorkflowConcurrency
	if concurrency > 0 {
		concurrencyOpts = worker.Concurrency(getConcurrencyKey).MaxRuns(int32(concurrency)) //nolint:gosec
	}
	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.Event("load-test:event"),
			Name:        "load-test",
			Description: "Load testing",
			Concurrency: concurrencyOpts,
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					l.Info().Msgf("executing %s", ctx.StepRunId())
					var input Event
					err = ctx.WorkflowInput(&input)
					if err != nil {
						return nil, err
					}

					took := time.Since(input.CreatedAt)
					l.Info().Msgf("executing %d took %s", input.ID, took)

					mx.Lock()
					executions <- took
					// detect duplicate in executed slice
					for i := 0; i < len(executed)-1; i++ {
						if executed[i] == input.ID {

							l.Error().Str("step-run-id", ctx.StepRunId()).Msgf("duplicate %d", input.ID)
							duplicateChan <- input.ID
							return nil, fmt.Errorf("duplicate %d", input.ID)

						}
					}

					uniques++

					executed = append(executed, input.ID)
					executedChan <- int64(input.ID)
					mx.Unlock()
					if delay > 0 {
						l.Info().Msgf("executed %d now delaying", input.ID)
						time.Sleep(delay)
						l.Info().Msgf("executed %d now done after %s", input.ID, delay)
					}
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

	cleanup, err := w.Start()

	if err != nil {
		panic(fmt.Errorf("error starting worker: %w", err))
	}
	defer func() {
		err := cleanup()
		if err != nil {
			panic(fmt.Errorf("error cleaning up worker: %w", err))
		}
	}()

	l.Info().Msg("worker started")
	<-ctx.Done()

	mx.Lock()
	defer mx.Unlock()
	l.Info().Msg("worker finished")
	return uniques
}
