package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
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

		workflowName := "cancellation"

		workflows, err := c.API().WorkflowListWithResponse(context.Background(), uuid.MustParse(c.TenantId()), &rest.WorkflowListParams{
			Name: &workflowName,
		})

		if err != nil {
			panic(fmt.Errorf("error listing workflows: %w", err))
		}

		if workflows.JSON200 == nil {
			panic(fmt.Errorf("no workflows found"))
		}

		rows := *workflows.JSON200.Rows

		if len(rows) == 0 {
			panic(fmt.Errorf("no workflows found"))
		}

		workflowId := uuid.MustParse(rows[0].Metadata.Id)

		workflowRuns, err := c.API().WorkflowRunListWithResponse(context.Background(), uuid.MustParse(c.TenantId()), &rest.WorkflowRunListParams{
			WorkflowId: &workflowId,
		})

		if err != nil {
			panic(fmt.Errorf("error listing workflow runs: %w", err))
		}

		if workflowRuns.JSON200 == nil {
			panic(fmt.Errorf("no workflow runs found"))
		}

		workflowRunsRows := *workflowRuns.JSON200.Rows

		_, err = c.API().WorkflowRunCancelWithResponse(context.Background(), uuid.MustParse(c.TenantId()), rest.WorkflowRunsCancelRequest{
			WorkflowRunIds: []uuid.UUID{uuid.MustParse(workflowRunsRows[0].Metadata.Id)},
		})

		if err != nil {
			panic(fmt.Errorf("error cancelling workflow run: %w", err))
		}
	}()

	cleanup, err := w.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting worker: %w", err)
	}

	return cleanup, nil
}
