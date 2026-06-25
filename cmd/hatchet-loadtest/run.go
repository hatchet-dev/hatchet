package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/features"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	v0worker "github.com/hatchet-dev/hatchet/pkg/worker"
)

type stepOneOutput struct {
	Message string `json:"message"`
}

type executionEvent struct {
	startedAt time.Time
	duration  time.Duration
}

func run(ctx context.Context, config LoadTestConfig, executions chan<- executionEvent, registered chan<- error) (int64, int64) {
	hatchet, err := v1.NewHatchetClient(
		v1.Config{
			Namespace: config.Namespace,
			Logger:    &l,
		},
	)

	if err != nil {
		panic(err)
	}

	mx := sync.Mutex{}
	var count int64
	var uniques int64
	var executed []int64

	step := func(ctx v0worker.HatchetContext, input Event) (any, error) {
		took := time.Since(input.CreatedAt)
		l.Info().Msgf("executing %d took %s", input.ID, took)

		mx.Lock()
		executions <- executionEvent{input.CreatedAt, took}
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

		time.Sleep(config.Delay)

		if config.FailureRate > 0 {
			if rand.Float32() < config.FailureRate { // nolint:gosec
				return nil, fmt.Errorf("random failure")
			}
		}

		return &stepOneOutput{
			Message: "This ran at: " + time.Now().Format(time.RFC3339Nano),
		}, nil
	}

	// put the rate limits
	for i := range config.RlKeys {
		err = hatchet.RateLimits().Upsert(
			features.CreateRatelimitOpts{
				// FIXME: namespace?
				Key:      "rl-key-" + fmt.Sprintf("%d", i),
				Limit:    config.RlLimit,
				Duration: types.RateLimitDuration(config.RlDurationUnit),
			},
		)

		if err != nil {
			panic(fmt.Errorf("error creating rate limit: %w", err))
		}
	}

	workflows := []workflow.WorkflowBase{}

	for i := range config.EventFanout {
		var concurrencyOpt []types.Concurrency

		if config.Concurrency > 0 {
			maxRuns := int32(config.Concurrency) // nolint: gosec
			limitStrategy := types.GroupRoundRobin

			concurrencyOpt = []types.Concurrency{
				{
					Expression:    "'global'",
					MaxRuns:       &maxRuns,
					LimitStrategy: &limitStrategy,
				},
			}
		}

		loadtest := factory.NewWorkflow[Event, stepOneOutput](
			create.WorkflowCreateOpts[Event]{
				Name: fmt.Sprintf("load-test-%d", i),
				OnEvents: []string{
					"load-test:event",
				},
				Concurrency: concurrencyOpt,
			},
			hatchet,
		)

		var prevTask *task.TaskDeclaration[Event]

		for j := range config.DagSteps {
			var parents []create.NamedTask

			if prevTask != nil {
				parentTask := prevTask
				parents = []create.NamedTask{
					parentTask,
				}
			}

			var rateLimits []*types.RateLimit

			if config.RlKeys > 0 {
				units := 1

				rateLimits = []*types.RateLimit{
					{
						Key:   fmt.Sprintf("rl-key-%d", i%config.RlKeys),
						Units: &units,
					},
				}
			}

			prevTask = loadtest.Task(
				create.WorkflowTask[Event, stepOneOutput]{
					Name:       fmt.Sprintf("step-%d", j),
					Parents:    parents,
					RateLimits: rateLimits,
				},
				step,
			)
		}

		workflows = append(workflows, loadtest)
	}

	worker, err := hatchet.Worker(
		worker.WorkerOpts{
			Name:      "load-test-worker",
			Workflows: workflows,
			Slots:     config.Slots,
			Logger:    &l,
		},
	)

	if err != nil {
		registered <- fmt.Errorf("error creating worker (workflow registration): %w", err)
		return 0, 0
	}

	registered <- nil

	if ctx.Err() != nil {
		return 0, 0
	}

	if config.WorkerDelay > 0 {
		l.Info().Msgf("waiting %s before starting the worker", config.WorkerDelay)
		time.Sleep(config.WorkerDelay)
	}

	l.Info().Msg("starting worker now")

	cleanup, err := worker.Start()
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
