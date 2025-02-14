package main

import (
	"context"
	"fmt"
	"math/rand/v2"
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

func run(ctx context.Context, delay time.Duration, executions chan<- time.Duration, concurrency, slots int, failureRate float32) (int64, int64) {
	c, err := client.New(
		client.WithLogLevel("warn"),
	)

	if err != nil {
		panic(err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
		worker.WithLogLevel("warn"),
		worker.WithMaxRuns(slots),
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

	step := func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
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
		var duplicate bool
		// for i := 0; i < len(executed)-1; i++ {
		// 	if executed[i] == input.ID {
		// 		duplicate = true
		// 		break
		// 	}
		// }
		if duplicate {
			l.Warn().Str("step-run-id", ctx.StepRunId()).Msgf("duplicate %d", input.ID)
		}
		if !duplicate {
			uniques++
		}
		count++
		executed = append(executed, input.ID)
		mx.Unlock()

		time.Sleep(delay)

		if failureRate > 0 {
			if rand.Float32() < failureRate {
				return nil, fmt.Errorf("random failure")
			}
		}

		return &stepOneOutput{
			Message: "This ran at: " + time.Now().Format(time.RFC3339Nano),
		}, nil
	}

	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			Name:        "load-test-1",
			Description: "Load testing",
			On:          worker.Event("load-test:event"),
			Concurrency: concurrencyOpts,
			// ScheduleTimeout: "30s",
			Steps: []*worker.WorkflowStep{
				worker.Fn(step).SetName("step-one").SetTimeout("5m"),
			},
		},
	)

	if err != nil {
		panic(err)
	}

	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			Name:        "load-test-2",
			Description: "Load testing",
			On:          worker.Event("load-test:event"),
			Concurrency: concurrencyOpts,
			// ScheduleTimeout: "30s",
			Steps: []*worker.WorkflowStep{
				worker.Fn(step).SetName("step-one").SetTimeout("5m"),
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

	<-ctx.Done()

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}

	mx.Lock()
	defer mx.Unlock()
	return count, uniques
}
