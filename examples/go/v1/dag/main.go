package main

import (
	"context"
	"log"

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
	step1 := workflow.AddTask("step-1", func(ctx hatchet.Context, input Input) (StepOutput, error) {
		return StepOutput{
			Step:   1,
			Result: input.Value * 2,
		}, nil
	})

	// Step 2: Depends on step 1
	step2 := workflow.AddTask("step-2", func(ctx hatchet.Context, input Input) (StepOutput, error) {
		// Get output from step 1
		var step1Output StepOutput
		if err := ctx.ParentOutput(step1.NamedTask, &step1Output); err != nil {
			return StepOutput{}, err
		}

		return StepOutput{
			Step:   2,
			Result: step1Output.Result + 10,
		}, nil
	}, hatchet.WithParents(step1.NamedTask))

	// Step 3: Also depends on step 1, parallel to step 2
	step3 := workflow.AddTask("step-3", func(ctx hatchet.Context, input Input) (StepOutput, error) {
		// Get output from step 1
		var step1Output StepOutput
		if err := ctx.ParentOutput(step1.NamedTask, &step1Output); err != nil {
			return StepOutput{}, err
		}

		return StepOutput{
			Step:   3,
			Result: step1Output.Result * 3,
		}, nil
	}, hatchet.WithParents(step1.NamedTask))

	// Final step: Combines outputs from step 2 and step 3
	workflow.AddTask("final-step", func(ctx hatchet.Context, input Input) (StepOutput, error) {
		var step2Output, step3Output StepOutput

		if err := ctx.ParentOutput(step2.NamedTask, &step2Output); err != nil {
			return StepOutput{}, err
		}
		if err := ctx.ParentOutput(step3.NamedTask, &step3Output); err != nil {
			return StepOutput{}, err
		}

		return StepOutput{
			Step:   4,
			Result: step2Output.Result + step3Output.Result,
		}, nil
	}, hatchet.WithParents(step2.NamedTask, step3.NamedTask))

	worker, err := client.NewWorker("dag-worker", hatchet.WithWorkflows(workflow))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Run the workflow
	err = client.Run(context.Background(), "dag-workflow", Input{Value: 5})
	if err != nil {
		log.Fatalf("failed to run workflow: %v", err)
	}

	if err := worker.StartBlocking(context.Background()); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
