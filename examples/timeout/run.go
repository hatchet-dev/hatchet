package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func run(done chan<- string, job worker.WorkflowJob) (func() error, error) {
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

	err = w.On(
		worker.Events("user:create:timeout"),
		&job,
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
			"user:create:timeout",
			testEvent,
		)
		if err != nil {
			panic(fmt.Errorf("error pushing event: %w", err))
		}

		time.Sleep(20 * time.Second)

		client := db.NewClient()
		if err := client.Connect(); err != nil {
			panic(fmt.Errorf("error connecting to database: %w", err))
		}
		defer client.Disconnect()

		// TODO check for the database status

		events, err := client.Event.FindMany(
			db.Event.TenantID.Equals(c.TenantId()),
			db.Event.Key.Equals("user:create:timeout"),
		).With(
			db.Event.WorkflowRuns.Fetch().With(
				db.WorkflowRunTriggeredBy.Parent.Fetch().With(
					db.WorkflowRun.JobRuns.Fetch().With(
						db.JobRun.StepRuns.Fetch(),
					),
				),
			),
		).Exec(context.Background())
		if err != nil {
			panic(fmt.Errorf("error finding events: %w", err))
		}

		for _, event := range events {
			for _, workflowRun := range event.WorkflowRuns() {
				for _, jobRuns := range workflowRun.Parent().JobRuns() {
					for _, stepRun := range jobRuns.StepRuns() {
						if stepRun.Status != db.StepRunStatusFailed {
							panic(fmt.Errorf("expected step run to be failed, got %s", stepRun.Status))
						}
					}
				}
			}
		}

		done <- "done"
	}()

	cleanup, err := w.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting worker: %w", err)
	}

	return cleanup, nil
}
