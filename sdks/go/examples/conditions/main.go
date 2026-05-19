package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type StepOutput struct {
	RandomNumber int `json:"random_number"`
}

type RandomSum struct {
	Sum int `json:"sum"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Create a workflow
	workflow := client.NewWorkflow("TaskConditionWorkflow")
	// !!

	// > Add base task
	start := workflow.NewTask("start", func(ctx hatchet.Context, _ any) (StepOutput, error) {
		return StepOutput{RandomNumber: rand.Intn(100) + 1}, nil //nolint:gosec
	})
	// !!

	// > Add wait for sleep
	waitForSleep := workflow.NewTask("wait-for-sleep", func(ctx hatchet.Context, _ any) (StepOutput, error) {
		return StepOutput{RandomNumber: rand.Intn(100) + 1}, nil //nolint:gosec
	},
		hatchet.WithParents(start),
		hatchet.WithWaitFor(hatchet.SleepCondition(10*time.Second)),
	)
	// !!

	// > Add skip condition override
	_ = workflow.NewTask("skip-with-multiple-parents", func(ctx hatchet.Context, _ any) (StepOutput, error) {
		return StepOutput{RandomNumber: rand.Intn(100) + 1}, nil //nolint:gosec
	},
		hatchet.WithParents(start, waitForSleep),
		hatchet.WithSkipIf(hatchet.ParentCondition(start, "output.random_number > 0")),
	)
	// !!

	// > Add skip on event
	skipOnEvent := workflow.NewTask("skip-on-event", func(ctx hatchet.Context, _ any) (StepOutput, error) {
		return StepOutput{RandomNumber: rand.Intn(100) + 1}, nil //nolint:gosec
	},
		hatchet.WithParents(start),
		hatchet.WithWaitFor(hatchet.SleepCondition(30*time.Second)),
		hatchet.WithSkipIf(hatchet.UserEventCondition("skip_on_event:skip", "")),
	)
	// !!

	// > Add branching
	leftBranch := workflow.NewTask("left-branch", func(ctx hatchet.Context, _ any) (StepOutput, error) {
		return StepOutput{RandomNumber: rand.Intn(100) + 1}, nil //nolint:gosec
	},
		hatchet.WithParents(waitForSleep),
		hatchet.WithSkipIf(hatchet.ParentCondition(waitForSleep, "output.random_number > 50")),
	)

	rightBranch := workflow.NewTask("right-branch", func(ctx hatchet.Context, _ any) (StepOutput, error) {
		return StepOutput{RandomNumber: rand.Intn(100) + 1}, nil //nolint:gosec
	},
		hatchet.WithParents(waitForSleep),
		hatchet.WithSkipIf(hatchet.ParentCondition(waitForSleep, "output.random_number <= 50")),
	)
	// !!

	// > Add wait for event
	waitForEvent := workflow.NewTask("wait-for-event", func(ctx hatchet.Context, _ any) (StepOutput, error) {
		return StepOutput{RandomNumber: rand.Intn(100) + 1}, nil //nolint:gosec
	},
		hatchet.WithParents(start),
		hatchet.WithWaitFor(hatchet.OrCondition(
			hatchet.SleepCondition(1*time.Minute),
			hatchet.UserEventCondition("wait_for_event:start", ""),
		)),
	)
	// !!

	// > Add sum
	_ = workflow.NewTask("sum", func(ctx hatchet.Context, _ any) (RandomSum, error) {
		var startOut StepOutput
		err := ctx.ParentOutput(start, &startOut)
		if err != nil {
			return RandomSum{}, err
		}

		var waitForEventOut StepOutput
		err = ctx.ParentOutput(waitForEvent, &waitForEventOut)
		if err != nil {
			return RandomSum{}, err
		}

		var waitForSleepOut StepOutput
		err = ctx.ParentOutput(waitForSleep, &waitForSleepOut)
		if err != nil {
			return RandomSum{}, err
		}

		total := startOut.RandomNumber + waitForEventOut.RandomNumber + waitForSleepOut.RandomNumber

		if !ctx.WasSkipped(skipOnEvent) {
			var out StepOutput
			err = ctx.ParentOutput(skipOnEvent, &out)
			if err == nil {
				total += out.RandomNumber
			}
		}

		if !ctx.WasSkipped(leftBranch) {
			var out StepOutput
			err = ctx.ParentOutput(leftBranch, &out)
			if err == nil {
				total += out.RandomNumber
			}
		}

		if !ctx.WasSkipped(rightBranch) {
			var out StepOutput
			err = ctx.ParentOutput(rightBranch, &out)
			if err == nil {
				total += out.RandomNumber
			}
		}

		return RandomSum{Sum: total}, nil
	}, hatchet.WithParents(
		start,
		waitForSleep,
		waitForEvent,
		skipOnEvent,
		leftBranch,
		rightBranch,
	))
	// !!

	worker, err := client.NewWorker("dag-worker", hatchet.WithWorkflows(workflow))
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

	_, err = client.Run(context.Background(), "TaskConditionWorkflow", nil)
	if err != nil {
		log.Fatalf("failed to run workflow: %v", err)
	}

	<-interruptCtx.Done()
}
