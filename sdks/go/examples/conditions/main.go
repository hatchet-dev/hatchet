package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type WorkflowInput struct {
	ProcessID string `json:"process_id"`
}

type StepOutput struct {
	StepName     string `json:"step_name"`
	ProcessedAt  string `json:"processed_at"`
	RandomNumber int    `json:"random_number"`
}

type SumOutput struct {
	Summary string `json:"summary"`
	Total   int    `json:"total"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Create a workflow
	workflow := client.NewWorkflow("conditional-workflow")
	// !!

	// Initial task
	// > Add base task
	start := workflow.NewTask("start", func(ctx hatchet.Context, input WorkflowInput) (StepOutput, error) {
		randomNum := rand.Intn(100) + 1 //nolint:gosec // This is a demo
		log.Printf("Starting workflow for process %s with random number: %d", input.ProcessID, randomNum)

		return StepOutput{
			StepName:     "start",
			RandomNumber: randomNum,
			ProcessedAt:  time.Now().Format(time.RFC3339),
		}, nil
	})
	// !!

	// > Add wait for sleep
	waitForSleep := workflow.NewTask("wait-for-sleep", func(ctx hatchet.Context, input WorkflowInput) (StepOutput, error) {
		return StepOutput{
			RandomNumber: rand.Intn(100) + 1,
		}, nil
	},
		hatchet.WithParents(start),
		hatchet.WithWaitFor(hatchet.SleepCondition(10*time.Second)),
	)
	// !!

	// > Add skip on event
	// Task that waits for either 10 seconds or a user event
	skipOnEvent := workflow.NewTask("skip-on-event", func(ctx hatchet.Context, input WorkflowInput) (StepOutput, error) {
		log.Printf("Skip on event task completed for process %s", input.ProcessID)
		return StepOutput{
			StepName:     "skip-on-event",
			RandomNumber: rand.Intn(50) + 1, //nolint:gosec // This is a demo
			ProcessedAt:  time.Now().Format(time.RFC3339),
		}, nil
	},
		hatchet.WithParents(start),
		hatchet.WithWaitFor(hatchet.SleepCondition(10*time.Second)),
		hatchet.WithSkipIf(hatchet.UserEventCondition("process:skip", "true")),
	)
	// !!

	// > Add wait for event
	// Task that might be skipped based on external event
	skipableTask := workflow.NewTask("skipable-task", func(ctx hatchet.Context, input WorkflowInput) (StepOutput, error) {
		log.Printf("Skipable task executing for process %s", input.ProcessID)
		return StepOutput{
			StepName:     "skipable-task",
			RandomNumber: rand.Intn(10) + 1, //nolint:gosec // This is a demo
			ProcessedAt:  time.Now().Format(time.RFC3339),
		}, nil
	},
		hatchet.WithParents(start),
		hatchet.WithWaitFor(hatchet.SleepCondition(3*time.Second)),
		hatchet.WithSkipIf(hatchet.UserEventCondition("process:skip", "true")),
	)
	// !!

	// > Add branching
	// Left branch - only runs if start's random number <= 50
	leftBranch := workflow.NewTask("left-branch", func(ctx hatchet.Context, input WorkflowInput) (StepOutput, error) {
		log.Printf("Left branch executing for process %s", input.ProcessID)
		return StepOutput{
			StepName:     "left-branch",
			RandomNumber: rand.Intn(25) + 1, //nolint:gosec // This is a demo
			ProcessedAt:  time.Now().Format(time.RFC3339),
		}, nil
	},
		hatchet.WithParents(waitForSleep),
		hatchet.WithSkipIf(hatchet.ParentCondition(start, "output.randomNumber > 50")),
	)

	// Right branch - only runs if start's random number > 50
	rightBranch := workflow.NewTask("right-branch", func(ctx hatchet.Context, input WorkflowInput) (StepOutput, error) {
		log.Printf("Right branch executing for process %s", input.ProcessID)
		return StepOutput{
			StepName:     "right-branch",
			RandomNumber: rand.Intn(25) + 26, //nolint:gosec // This is a demo
			ProcessedAt:  time.Now().Format(time.RFC3339),
		}, nil
	},
		hatchet.WithParents(waitForSleep),
		hatchet.WithSkipIf(hatchet.ParentCondition(start, "output.randomNumber <= 50")),
	)
	// !!

	// Final aggregation task
	// > Add sum
	_ = workflow.NewTask("summarize", func(ctx hatchet.Context, input WorkflowInput) (SumOutput, error) {
		var total int
		var summary string

		// Get start output
		var startOutput StepOutput
		if err := ctx.ParentOutput(start, &startOutput); err != nil {
			return SumOutput{}, err
		}
		total += startOutput.RandomNumber
		summary = fmt.Sprintf("Start: %d", startOutput.RandomNumber)

		// Get wait for sleep output
		var waitForSleepOutput StepOutput
		if err := ctx.ParentOutput(waitForSleep, &waitForSleepOutput); err != nil {
			return SumOutput{}, err
		}
		total += waitForSleepOutput.RandomNumber
		summary += fmt.Sprintf(", Wait for sleep: %d", waitForSleepOutput.RandomNumber)

		// Get skip on event output
		var skipOnEventOutput StepOutput
		if err := ctx.ParentOutput(skipOnEvent, &skipOnEventOutput); err != nil {
			return SumOutput{}, err
		}
		total += skipOnEventOutput.RandomNumber
		summary += fmt.Sprintf(", Skip on event: %d", skipOnEventOutput.RandomNumber)

		// Try to get left branch output (might be skipped)
		var leftOutput StepOutput
		if err := ctx.ParentOutput(leftBranch, &leftOutput); err == nil {
			total += leftOutput.RandomNumber
			summary += fmt.Sprintf(", Left: %d", leftOutput.RandomNumber)
		} else {
			summary += ", Left: skipped"
		}

		// Try to get right branch output (might be skipped)
		var rightOutput StepOutput
		if err := ctx.ParentOutput(rightBranch, &rightOutput); err == nil {
			total += rightOutput.RandomNumber
			summary += fmt.Sprintf(", Right: %d", rightOutput.RandomNumber)
		} else {
			summary += ", Right: skipped"
		}

		// Try to get skipable task output (might be skipped)
		var skipableOutput StepOutput
		if err := ctx.ParentOutput(skipableTask, &skipableOutput); err == nil {
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
		start,
		waitForSleep,
		skipOnEvent,
		leftBranch,
		rightBranch,
		skipableTask,
	))
	// !!

	worker, err := client.NewWorker("conditional-worker", hatchet.WithWorkflows(workflow))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	go func() {
		log.Println("Starting conditional workflow worker...")
		if err := worker.StartBlocking(interruptCtx); err != nil {
			log.Fatalf("failed to start worker: %v", err)
		}
	}()

	// Run the workflow
	_, err = client.Run(context.Background(), "conditional-workflow", WorkflowInput{
		ProcessID: "demo-process-1",
	})
	if err != nil {
		log.Fatalf("failed to run workflow: %v", err)
	}

	<-interruptCtx.Done()
}
