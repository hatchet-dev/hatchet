package main

import (
	"fmt"
	"log"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func runBulk() (func() error, error) {
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
			On:             worker.Events("user:create:sticky"),
			Name:           "sticky2",
			Description:    "sticky",
			StickyStrategy: types.StickyStrategyPtr(types.StickyStrategy_SOFT),
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {

					sticky := true

					_, err = ctx.SpawnWorkflows(
						[]*worker.SpawnWorkflowsOpts{
							{
								WorkflowName: "step-one",

								Sticky: &sticky,
							},
							{
								WorkflowName: "step-one",

								Sticky: &sticky,
							},
						})

					if err != nil {
						return nil, fmt.Errorf("error spawning workflow: %w", err)
					}

					return &stepOneOutput{
						Message: ctx.Worker().ID(),
					}, nil
				}).SetName("step-one"),
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					return &stepOneOutput{
						Message: ctx.Worker().ID(),
					}, nil
				}).SetName("step-two"),
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					return &stepOneOutput{
						Message: ctx.Worker().ID(),
					}, nil
				}).SetName("step-three").AddParents("step-one", "step-two"),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}

	go func() {
		log.Printf("pushing workflow")

		var workflows []*client.WorkflowRun
		for i := 0; i < 100; i++ {
			data := map[string]interface{}{
				"username": fmt.Sprintf("echo-test-%d", i),
				"user_id":  fmt.Sprintf("1234-%d", i),
			}
			workflows = append(workflows, &client.WorkflowRun{
				Name:  "sticky2",
				Input: data,
				Options: []client.RunOptFunc{
					// setting a dedupe key so these shouldn't all run
					client.WithRunMetadata(map[string]interface{}{
						// "dedupe": "dedupe1",
					}),
				},
			})

		}

		outs, err := c.Admin().BulkRunWorkflow(workflows)
		if err != nil {
			panic(fmt.Errorf("error pushing event: %w", err))
		}

		for _, out := range outs {
			log.Printf("workflow run id: %v", out)
		}

	}()

	cleanup, err := w.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting worker: %w", err)
	}

	return cleanup, nil
}

func runSingles() (func() error, error) {
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
			On:             worker.Events("user:create:sticky"),
			Name:           "sticky",
			Description:    "sticky",
			StickyStrategy: types.StickyStrategyPtr(types.StickyStrategy_SOFT),
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {

					sticky := true

					_, err = ctx.SpawnWorkflow("step-one", nil, &worker.SpawnWorkflowOpts{
						Sticky: &sticky,
					})

					if err != nil {
						return nil, fmt.Errorf("error spawning workflow: %w", err)
					}

					return &stepOneOutput{
						Message: ctx.Worker().ID(),
					}, nil
				}).SetName("step-one"),
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					return &stepOneOutput{
						Message: ctx.Worker().ID(),
					}, nil
				}).SetName("step-two"),
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					return &stepOneOutput{
						Message: ctx.Worker().ID(),
					}, nil
				}).SetName("step-three").AddParents("step-one", "step-two"),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}

	log.Printf("pushing workflow")

	var workflows []*client.WorkflowRun
	for i := 0; i < 100; i++ {
		data := map[string]interface{}{
			"username": fmt.Sprintf("echo-test-%d", i),
			"user_id":  fmt.Sprintf("1234-%d", i),
		}
		workflows = append(workflows, &client.WorkflowRun{
			Name:  "sticky",
			Input: data,
			Options: []client.RunOptFunc{
				client.WithRunMetadata(map[string]interface{}{
					// "dedupe": "dedupe1",
				}),
			},
		})
	}

	for _, wf := range workflows {

		go func() {
			out, err := c.Admin().RunWorkflow(wf.Name, wf.Input, wf.Options...)
			if err != nil {
				panic(fmt.Errorf("error pushing event: %w", err))
			}

			log.Printf("workflow run id: %v", out)
		}()

	}

	cleanup, err := w.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting worker: %w", err)
	}

	return cleanup, nil
}
