package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type concurrencyLimitEvent struct {
	ConcurrencyKey string `json:"concurrency_key"`
	UserId         int    `json:"user_id"`
}

type stepOneOutput struct {
	Message                 string `json:"message"`
	ConcurrencyWhenFinished int    `json:"concurrency_when_finished"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	ctx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := run(ctx); err != nil {
		panic(err)
	}
}

func getConcurrencyKey(ctx worker.HatchetContext) (string, error) {
	return "concurrency", nil
}

var done = make(chan struct{})
var errChan = make(chan error)

var workflowCount int
var countMux sync.Mutex

func run(ctx context.Context) error {
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

	// runningCount := 0

	countMux := sync.Mutex{}

	var countMap = make(map[string]int)
	maxConcurrent := 2

	err = w.RegisterWorkflow(

		&worker.WorkflowJob{
			Name:        "concurrency-limit-round-robin-existing-workflows",
			Description: "This limits concurrency to maxConcurrent runs at a time.",
			On:          worker.Events("test:concurrency-limit-round-robin-existing-workflows"),
			Concurrency: worker.Expression("input.concurrency_key").MaxRuns(int32(maxConcurrent)).LimitStrategy(types.GroupRoundRobin),

			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					input := &concurrencyLimitEvent{}

					err = ctx.WorkflowInput(input)

					if err != nil {
						return nil, fmt.Errorf("error getting input: %w", err)
					}
					concurrencyKey := input.ConcurrencyKey
					countMux.Lock()

					if countMap[concurrencyKey]+1 > maxConcurrent {
						countMux.Unlock()
						e := fmt.Errorf("concurrency limit exceeded for %d we have %d workers running", input.UserId, countMap[concurrencyKey])
						errChan <- e
						return nil, e
					}
					countMap[concurrencyKey]++

					countMux.Unlock()

					fmt.Println("received event", input.UserId)

					time.Sleep(10 * time.Second)

					fmt.Println("processed event", input.UserId)

					countMux.Lock()
					countMap[concurrencyKey]--
					countMux.Unlock()

					done <- struct{}{}

					return &stepOneOutput{}, nil
				},
				).SetName("step-one"),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error registering workflow: %w", err)
	}

	go func() {
		var workflowRuns []*client.WorkflowRun

		for i := 0; i < 1; i++ {
			workflowCount++
			event := concurrencyLimitEvent{
				ConcurrencyKey: "key",
				UserId:         i,
			}
			workflowRuns = append(workflowRuns, &client.WorkflowRun{
				Name:  "concurrency-limit-round-robin-existing-workflows",
				Input: event,
			})

		}

		// create a second one with a different key

		// so the bug we are testing here is that total concurrency for any one group should be 2
		// but if we have more than one group we end up with 4 running when only 2 + 1 are eligible to run

		for i := 0; i < 3; i++ {
			workflowCount++

			event := concurrencyLimitEvent{
				ConcurrencyKey: "secondKey",
				UserId:         i,
			}
			workflowRuns = append(workflowRuns, &client.WorkflowRun{
				Name:  "concurrency-limit-round-robin-existing-workflows",
				Input: event,
			})

		}

		_, err := c.Admin().BulkRunWorkflow(workflowRuns)
		if err != nil {
			fmt.Println("error running workflow", err)
		}

		fmt.Println("ran workflows")

	}()

	time.Sleep(2 * time.Second)
	cleanup, err := w.Start()
	if err != nil {
		return fmt.Errorf("error starting worker: %w", err)
	}
	defer cleanup()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(20 * time.Second):
			return fmt.Errorf("timeout")
		case err := <-errChan:
			return err
		case <-done:
			countMux.Lock()
			workflowCount--
			countMux.Unlock()
			if workflowCount == 0 {
				time.Sleep(1 * time.Second)
				return nil
			}

		}
	}
}
