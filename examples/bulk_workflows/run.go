package main

import (
	"fmt"
	"log"

	"github.com/hatchet-dev/hatchet/pkg/client"
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
			On:          worker.Events("user:create:simple"),
			Name:        "simple",
			Description: "This runs after an update to the user model.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					input := &userCreateEvent{}

					err = ctx.WorkflowInput(input)

					if err != nil {
						return nil, err
					}

					log.Printf("step-one")

					return &stepOneOutput{
						Message: "Username is: " + input.Username,
					}, nil
				},
				).SetName("step-one"),
				worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
					input := &stepOneOutput{}
					err = ctx.StepOutput("step-one", input)

					if err != nil {
						return nil, err
					}

					log.Printf("step-two")

					return &stepOneOutput{
						Message: "Above message is: " + input.Message,
					}, nil
				}).SetName("step-two").AddParents("step-one"),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}

	go func() {
		log.Printf("pushing workflow")

		var workflows []*client.WorkflowRun
		for i := 0; i < 999; i++ {
			data := map[string]interface{}{
				"username": fmt.Sprintf("echo-test-%d", i),
				"user_id":  fmt.Sprintf("1234-%d", i),
			}
			workflows = append(workflows, &client.WorkflowRun{
				Name:  "simple",
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

	if err != nil {
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}

	log.Printf("pushing workflow")

	var workflows []*client.WorkflowRun
	for i := 0; i < 999; i++ {
		data := map[string]interface{}{
			"username": fmt.Sprintf("echo-test-%d", i),
			"user_id":  fmt.Sprintf("1234-%d", i),
		}
		workflows = append(workflows, &client.WorkflowRun{
			Name:  "simple",
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
