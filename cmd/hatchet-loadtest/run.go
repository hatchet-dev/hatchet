package main

import (
	"context"
	"fmt"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client" //nolint
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	gosdk "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
	"math/rand/v2"
	"sync"
	"time"
)

type stepOneOutput struct {
	Message string `json:"message"`
}

type executionEvent struct {
	startedAt time.Time
	duration  time.Duration
}

func run(ctx context.Context, config LoadTestConfig, executions chan<- executionEvent) (int64, int64) {
	//nolint
	client, err := gosdk.NewClient(
		v0Client.WithNamespace(config.Namespace),
		v0Client.WithLogger(&l),
	)
	if err != nil {
		panic(err)
	}

	mx := sync.Mutex{}
	var count int64
	var uniques int64
	var executed []int64

	step := func(ctx gosdk.Context, input Event) (any, error) {
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
		err = client.RateLimits().Upsert(
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

	workflows := []gosdk.WorkflowBase{}

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

		loadtest := client.NewWorkflow(
			fmt.Sprintf("load-test-%d", i),
			gosdk.WithWorkflowEvents("load-test:event"),
			gosdk.WithWorkflowConcurrency(concurrencyOpt...),
		)

		var prevTask *gosdk.Task

		for j := range config.DagSteps {
			var parents []*gosdk.Task

			if prevTask != nil {
				parentTask := prevTask
				parents = []*gosdk.Task{
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

			prevTask = loadtest.NewTask(
				fmt.Sprintf("step-%d", j),
				step,
				gosdk.WithParents(parents...),
				gosdk.WithRateLimits(rateLimits...),
			)
		}

		workflows = append(workflows, loadtest)
	}

	worker, err := client.NewWorker(
		"load-test-worker",
		gosdk.WithWorkflows(workflows...),
		gosdk.WithSlots(config.Slots),
		gosdk.WithLogger(&l),
	)

	if err != nil {
		panic(fmt.Errorf("error creating worker: %w", err))
	}

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
