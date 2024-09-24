package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func run(events chan<- string) (func() error, error) {
	c, err := client.New()
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating worker: %w", err)
	}

	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.Events("user:create:cancellation"),
			Name:        "cancellation",
			Description: "cancellation",
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					select {
					case <-ctx.Done():
						events <- "done"
						log.Printf("context cancelled")
						return nil, nil
					case <-time.After(30 * time.Second):
						log.Printf("workflow never cancelled")
						return &stepOneOutput{
							Message: "done",
						}, nil
					}
				}).SetName("step-one"),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}

	go func() {
		log.Printf("pushing event")

		testEvent := userCreateEvent{
			Username: "echo-test",
			UserID:   "1234",
			Data: map[string]string{
				"test": "test",
			},
		}

		// push an event
		err := c.Event().Push(
			context.Background(),
			"user:create:cancellation",
			testEvent,
		)
		if err != nil {
			panic(fmt.Errorf("error pushing event: %w", err))
		}

		time.Sleep(10 * time.Second)

		//workflows, err := c.API().WorkflowListWithResponse(context.Background(), uuid.MustParse(c.TenantId()))
		//if err != nil {
		//	panic(fmt.Errorf("error listing workflows: %w", err))
		//}

		client := db.NewClient()
		if err := client.Connect(); err != nil {
			panic(fmt.Errorf("error connecting to database: %w", err))
		}
		defer client.Disconnect()

		stepRuns, err := client.StepRun.FindMany(
			db.StepRun.TenantID.Equals(c.TenantId()),
			db.StepRun.Status.Equals(db.StepRunStatusRunning),
		).Exec(context.Background())
		if err != nil {
			panic(fmt.Errorf("error finding step runs: %w", err))
		}

		if len(stepRuns) == 0 {
			panic(fmt.Errorf("no step runs to cancel"))
		}

		for _, stepRun := range stepRuns {
			stepRunID := stepRun.ID
			log.Printf("cancelling step run id: %s", stepRunID)
			res, err := c.API().StepRunUpdateCancelWithResponse(context.Background(), uuid.MustParse(c.TenantId()), uuid.MustParse(stepRunID))
			if err != nil {
				panic(fmt.Errorf("error cancelling step run: %w", err))
			}

			log.Printf("step run cancelled: %v", res.JSON200)
		}
	}()

	cleanup, err := w.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting worker: %w", err)
	}

	return cleanup, nil
}
