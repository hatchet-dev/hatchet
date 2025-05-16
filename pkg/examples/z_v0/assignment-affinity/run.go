package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func run() (func() error, error) {
	c, err := client.New()
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
		worker.WithLabels(map[string]interface{}{
			"model":  "fancy-ai-model-v2",
			"memory": 1024,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating worker: %w", err)
	}

	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.Events("user:create:affinity"),
			Name:        "affinity",
			Description: "affinity",
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {

					model := ctx.Worker().GetLabels()["model"]

					if model != "fancy-ai-model-v3" {
						ctx.Worker().UpsertLabels(map[string]interface{}{
							"model": nil,
						})
						// Do something to load the model
						ctx.Worker().UpsertLabels(map[string]interface{}{
							"model": "fancy-ai-model-v3",
						})
					}

					return &stepOneOutput{
						Message: ctx.Worker().ID(),
					}, nil
				}).
					SetName("step-one").
					SetDesiredLabels(map[string]*types.DesiredWorkerLabel{
						"model": {
							Value:  "fancy-ai-model-v3",
							Weight: 10,
						},
						"memory": {
							Value:      512,
							Required:   true,
							Comparator: types.ComparatorPtr(types.WorkerLabelComparator_GREATER_THAN),
						},
					}),
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
			"user:create:affinity",
			testEvent,
			nil,
			nil,
		)
		if err != nil {
			panic(fmt.Errorf("error pushing event: %w", err))
		}

		time.Sleep(10 * time.Second)
	}()

	cleanup, err := w.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting worker: %w", err)
	}

	return cleanup, nil
}
