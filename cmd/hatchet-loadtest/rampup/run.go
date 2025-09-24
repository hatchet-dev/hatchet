package rampup

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/rs/zerolog"
)

type stepOneOutput struct {
	Message string `json:"message"`
}

func run(ctx context.Context, delay time.Duration, concurrency int, maxAcceptableDuration time.Duration, hook chan<- time.Duration, executedCh chan<- int64) (int64, int64) {
	l := zerolog.New(os.Stderr).Level(zerolog.WarnLevel)

	c, err := hatchet.NewClient(
		client.WithLogger(&l),
	)

	if err != nil {
		panic(err)
	}

	mx := sync.Mutex{}
	var count int64
	var uniques int64
	var executed []int64

	var concurrencyOpts []types.Concurrency
	if concurrency > 0 {
		concurrencyOpts = []types.Concurrency{
			{
				Expression: "'my-key'",
				MaxRuns:    &[]int32{int32(concurrency)}[0],
			},
		}
	}

	task := c.NewStandaloneTask( // nolint: staticcheck
		"load-test",
		func(ctx hatchet.Context, input Event) (result *stepOneOutput, err error) {
			took := time.Since(input.CreatedAt)

			l.Debug().Msgf("executing %d took %s", input.ID, took)

			if took > maxAcceptableDuration {
				hook <- took
			}

			executedCh <- input.ID

			mx.Lock()

			// detect duplicate in executed slice
			var duplicate bool
			for i := 0; i < len(executed)-1; i++ {
				if executed[i] == input.ID {
					duplicate = true
				}
			}
			if duplicate {
				l.Warn().Str("step-run-id", ctx.StepRunId()).Msgf("duplicate %d", input.ID)
			} else {
				uniques++
			}
			count++
			executed = append(executed, input.ID)
			mx.Unlock()

			time.Sleep(delay)

			return &stepOneOutput{
				Message: "This ran at: " + time.Now().Format(time.RFC3339Nano),
			}, nil
		},
		hatchet.WithWorkflowDescription("Load testing"),
		hatchet.WithWorkflowEvents("load-test:event"),
		hatchet.WithWorkflowConcurrency(concurrencyOpts...),
	)

	w, err := c.NewWorker(
		"load-test-worker",
		hatchet.WithSlots(200),
		hatchet.WithLogger(&l),
		hatchet.WithWorkflows(task),
	)

	if err != nil {
		panic(err)
	}

	err = w.StartBlocking(ctx)
	if err != nil {
		panic(fmt.Errorf("error starting worker: %w", err))
	}

	mx.Lock()
	defer mx.Unlock()
	return count, uniques
}
