package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type WorkflowInput struct {
	ProcessID string `json:"process_id"`
}

type StepOutput struct {
	StepName     string `json:"step_name"`
	RandomNumber int    `json:"random_number"`
	ProcessedAt  string `json:"processed_at"`
}

type SumOutput struct {
	Total   int    `json:"total"`
	Summary string `json:"summary"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	workflow := client.NewWorkflow("conditional-workflow")

	// Initial task
	start := workflow.AddTask("start", func(ctx hatchet.Context, input WorkflowInput) (StepOutput, error) {
		randomNum := rand.Intn(100) + 1
		log.Printf("Starting workflow for process %s with random number: %d", input.ProcessID, randomNum)
		
		return StepOutput{
			StepName:     "start",
			RandomNumber: randomNum,
			ProcessedAt:  time.Now().Format(time.RFC3339),
		}, nil
	})

	// Task that waits for either 10 seconds or a user event
	waitForEvent := workflow.AddTask("wait-for-event", func(ctx hatchet.Context, input WorkflowInput) (StepOutput, error) {
		log.Printf("Wait task completed for process %s", input.ProcessID)
		return StepOutput{
			StepName:     "wait-for-event",
			RandomNumber: rand.Intn(50) + 1,
			ProcessedAt:  time.Now().Format(time.RFC3339),
		}, nil
	}, 
		hatchet.WithParents(start.NamedTask),
		hatchet.WithWaitFor(hatchet.OrCondition(
			hatchet.SleepCondition(10*time.Second),
			hatchet.UserEventCondition("process:continue", "true"),
		)),
	)

	// Left branch - only runs if start's random number <= 50
	leftBranch := workflow.AddTask("left-branch", func(ctx hatchet.Context, input WorkflowInput) (StepOutput, error) {
		log.Printf("Left branch executing for process %s", input.ProcessID)
		return StepOutput{
			StepName:     "left-branch",
			RandomNumber: rand.Intn(25) + 1,
			ProcessedAt:  time.Now().Format(time.RFC3339),
		}, nil
	}, 
		hatchet.WithParents(start.NamedTask),
		hatchet.WithSkipIf(hatchet.ParentCondition(start, "output.randomNumber > 50")),
	)

	// Right branch - only runs if start's random number > 50
	rightBranch := workflow.AddTask("right-branch", func(ctx hatchet.Context, input WorkflowInput) (StepOutput, error) {
		log.Printf("Right branch executing for process %s", input.ProcessID)
		return StepOutput{
			StepName:     "right-branch", 
			RandomNumber: rand.Intn(25) + 26,
			ProcessedAt:  time.Now().Format(time.RFC3339),
		}, nil
	}, 
		hatchet.WithParents(start.NamedTask),
		hatchet.WithSkipIf(hatchet.ParentCondition(start, "output.randomNumber <= 50")),
	)

	// Task that might be skipped based on external event
	skipableTask := workflow.AddTask("skipable-task", func(ctx hatchet.Context, input WorkflowInput) (StepOutput, error) {
		log.Printf("Skipable task executing for process %s", input.ProcessID)
		return StepOutput{
			StepName:     "skipable-task",
			RandomNumber: rand.Intn(10) + 1,
			ProcessedAt:  time.Now().Format(time.RFC3339),
		}, nil
	}, 
		hatchet.WithParents(start.NamedTask),
		hatchet.WithWaitFor(hatchet.SleepCondition(3*time.Second)),
		hatchet.WithSkipIf(hatchet.UserEventCondition("process:skip", "true")),
	)

	// Final aggregation task
	workflow.AddTask("summarize", func(ctx hatchet.Context, input WorkflowInput) (SumOutput, error) {
		var total int
		var summary string

		// Get start output
		var startOutput StepOutput
		if err := ctx.ParentOutput(start.NamedTask, &startOutput); err != nil {
			return SumOutput{}, err
		}
		total += startOutput.RandomNumber
		summary = fmt.Sprintf("Start: %d", startOutput.RandomNumber)

		// Get wait output
		var waitOutput StepOutput
		if err := ctx.ParentOutput(waitForEvent.NamedTask, &waitOutput); err != nil {
			return SumOutput{}, err
		}
		total += waitOutput.RandomNumber
		summary += fmt.Sprintf(", Wait: %d", waitOutput.RandomNumber)

		// Try to get left branch output (might be skipped)
		var leftOutput StepOutput
		if err := ctx.ParentOutput(leftBranch.NamedTask, &leftOutput); err == nil {
			total += leftOutput.RandomNumber
			summary += fmt.Sprintf(", Left: %d", leftOutput.RandomNumber)
		} else {
			summary += ", Left: skipped"
		}

		// Try to get right branch output (might be skipped)
		var rightOutput StepOutput
		if err := ctx.ParentOutput(rightBranch.NamedTask, &rightOutput); err == nil {
			total += rightOutput.RandomNumber
			summary += fmt.Sprintf(", Right: %d", rightOutput.RandomNumber)
		} else {
			summary += ", Right: skipped"
		}

		// Try to get skipable task output (might be skipped)
		var skipableOutput StepOutput
		if err := ctx.ParentOutput(skipableTask.NamedTask, &skipableOutput); err == nil {
			total += skipableOutput.RandomNumber
			summary += fmt.Sprintf(", Skipable: %d", skipableOutput.RandomNumber)
		} else {
			summary += ", Skipable: skipped"
		}

		log.Printf("Final summary for process %s: total=%d, %s", input.ProcessID, total, summary)

		return SumOutput{
			Total:   total,
			Summary: summary,
		}, nil
	}, hatchet.WithParents(
		waitForEvent.NamedTask,
		leftBranch.NamedTask,
		rightBranch.NamedTask,
		skipableTask.NamedTask,
	))

	worker, err := client.NewWorker("conditional-worker", hatchet.WithWorkflows(workflow))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Run the workflow
	err = client.Run(context.Background(), "conditional-workflow", WorkflowInput{
		ProcessID: "demo-process-1",
	})
	if err != nil {
		log.Fatalf("failed to run workflow: %v", err)
	}

	log.Println("Starting conditional workflow worker...")
	if err := worker.StartBlocking(context.Background()); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}