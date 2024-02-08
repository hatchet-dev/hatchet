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

func run(ctx context.Context) (int64, int64) {
	c, err := client.New()

	if err != nil {
		panic(err)
	}

	// Create a worker. This automatically reads in a TemporalClient from .env and workflow files from the .hatchet
	// directory, but this can be customized with the `worker.WithTemporalClient` and `worker.WithWorkflowFiles` options.
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

	err = w.On(
		worker.Event("test:event"),
		&worker.WorkflowJob{
			Name:        "scheduled-workflow",
			Description: "This runs at a scheduled time.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					var input Event
					err = ctx.WorkflowInput(&input)
					if err != nil {
						return nil, err
					}

					fmt.Println(input.ID, "delay", time.Since(input.CreatedAt))

					mx.Lock()
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
