package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/loadtest/eventkeys"
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

// executionKey identifies one task run the load test expects exactly once: the
// action (workflow and step, so the fanout copies and the DAG steps of one
// event are distinct runs) and the event that triggered it.
type executionKey struct {
	action  string
	eventID int64
}

// executionTally counts task executions and how many of them were the first
// for their key. The tool pushes every event once and configures no retries,
// so a second execution of a key is the engine dispatching a task run twice;
// it is reported as a duplicate instead of being folded into the executed
// count, where it would mask a lost event.
type executionTally struct {
	mu      sync.Mutex
	count   int64
	uniques int64
	seen    map[executionKey]struct{}
}

func newExecutionTally() *executionTally {
	return &executionTally{seen: map[executionKey]struct{}{}}
}

// record counts one execution and reports whether its key was executed before.
func (t *executionTally) record(key executionKey) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.count++
	if _, dup := t.seen[key]; dup {
		return true
	}
	t.seen[key] = struct{}{}
	t.uniques++

	return false
}

// counts returns the executions so far and how many distinct keys they cover.
func (t *executionTally) counts() (count, uniques int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.count, t.uniques
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

	tally := newExecutionTally()

	step := func(ctx v0worker.HatchetContext, input Event) (any, error) {
		took := time.Since(input.CreatedAt)
		l.Info().Msgf("executing %d took %s", input.ID, took)

		executions <- executionEvent{input.CreatedAt, took}

		key := executionKey{action: ctx.ActionId(), eventID: input.ID}
		if tally.record(key) {
			l.Warn().Str("step-run-id", ctx.StepRunId()).Str("action", key.action).Msgf("duplicate execution of event %d", input.ID)
		}

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
				Name: eventkeys.WorkflowStandardName(i),
				OnEvents: []string{
					eventkeys.EventKeyDefault.String(),
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
			Name:      eventkeys.WorkerName,
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

	return tally.counts()
}
