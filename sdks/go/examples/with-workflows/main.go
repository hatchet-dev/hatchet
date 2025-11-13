// Code contributed by @jnfrati as part of https://github.com/hatchet-dev/hatchet/issues/2356

package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type Input struct {
	Value int `json:"value"`
}

type StepOutput struct {
	Step   int `json:"step"`
	Result int `json:"result"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create a DAG workflow
	workflow := client.NewWorkflow("dag-workflow")

	// Step 1: Initial processing
	step1 := workflow.NewTask("step-1", func(ctx hatchet.Context, input Input) (StepOutput, error) {
		return StepOutput{
			Step:   1,
			Result: input.Value * 2,
		}, nil
	})

	// Step 2: Depends on step 1
	step2 := workflow.NewTask("step-2", func(ctx hatchet.Context, input Input) (StepOutput, error) {
		// Get output from step 1
		var step1Output StepOutput
		if err := ctx.ParentOutput(step1, &step1Output); err != nil {
			return StepOutput{}, err
		}

		return StepOutput{
			Step:   2,
			Result: step1Output.Result + 10,
		}, nil
	}, hatchet.WithParents(step1))

	// Step 3: Also depends on step 1, parallel to step 2
	step3 := workflow.NewTask("step-3", func(ctx hatchet.Context, input Input) (StepOutput, error) {
		// Get output from step 1
		var step1Output StepOutput
		if err := ctx.ParentOutput(step1, &step1Output); err != nil {
			return StepOutput{}, err
		}

		return StepOutput{
			Step:   3,
			Result: step1Output.Result * 3,
		}, nil
	}, hatchet.WithParents(step1))

	// Final step: Combines outputs from step 2 and step 3
	_ = workflow.NewTask("final-step", func(ctx hatchet.Context, input Input) (StepOutput, error) {
		var step2Output, step3Output StepOutput

		if err := ctx.ParentOutput(step2, &step2Output); err != nil {
			return StepOutput{}, err
		}
		if err := ctx.ParentOutput(step3, &step3Output); err != nil {
			return StepOutput{}, err
		}

		_ = client.Events().Push(ctx, "dag-workflow", map[string]any{})

		return StepOutput{
			Step:   4,
			Result: step2Output.Result + step3Output.Result,
		}, nil
	}, hatchet.WithParents(step2, step3))

	eventWorkflow := client.NewWorkflow(
		"event-workflow",
		hatchet.WithWorkflowEvents("dag-workflow"),
	)

	eventWorkflow.NewTask("log-event", func(ctx hatchet.Context, input any) (any, error) {
		log.Printf("Received event: %v", input)
		return nil, nil
	})

	workflowArr := []hatchet.WorkflowBase{workflow, eventWorkflow}

	worker, err := client.NewWorker(
		"dag-worker",
		hatchet.WithWorkflows(
			workflowArr...,
		))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Printf("failed to start worker: %v", err)
	}
}
