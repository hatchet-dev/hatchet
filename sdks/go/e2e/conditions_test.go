//go:build e2e

package e2e

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type ConditionStepOutput struct {
	RandomNumber int `json:"random_number"`
}

type ConditionSumOutput struct {
	Sum int `json:"sum"`
}

func TestConditions(t *testing.T) {
	client := newClient(t)

	workflow := client.NewWorkflow("conditions-e2e")

	start := workflow.NewTask("start", func(ctx hatchet.Context, input any) (ConditionStepOutput, error) {
		return ConditionStepOutput{RandomNumber: rand.Intn(100) + 1}, nil //nolint:gosec
	})

	waitForSleep := workflow.NewTask("wait-for-sleep", func(ctx hatchet.Context, input any) (ConditionStepOutput, error) {
		return ConditionStepOutput{RandomNumber: rand.Intn(100) + 1}, nil //nolint:gosec
	},
		hatchet.WithParents(start),
		hatchet.WithWaitFor(hatchet.SleepCondition(10*time.Second)),
	)

	skipOnEvent := workflow.NewTask("skip-on-event", func(ctx hatchet.Context, input any) (ConditionStepOutput, error) {
		return ConditionStepOutput{RandomNumber: rand.Intn(50) + 1}, nil //nolint:gosec
	},
		hatchet.WithParents(start),
		hatchet.WithWaitFor(hatchet.SleepCondition(30*time.Second)),
		hatchet.WithSkipIf(hatchet.UserEventCondition("conditions-e2e:skip", "true")),
	)

	leftBranch := workflow.NewTask("left-branch", func(ctx hatchet.Context, input any) (ConditionStepOutput, error) {
		return ConditionStepOutput{RandomNumber: rand.Intn(100) + 1}, nil //nolint:gosec
	},
		hatchet.WithParents(waitForSleep),
		hatchet.WithSkipIf(hatchet.ParentCondition(start, "output.random_number > 50")),
	)

	rightBranch := workflow.NewTask("right-branch", func(ctx hatchet.Context, input any) (ConditionStepOutput, error) {
		return ConditionStepOutput{RandomNumber: rand.Intn(100) + 1}, nil //nolint:gosec
	},
		hatchet.WithParents(waitForSleep),
		hatchet.WithSkipIf(hatchet.ParentCondition(start, "output.random_number <= 50")),
	)

	waitForEvent := workflow.NewTask("wait-for-event", func(ctx hatchet.Context, input any) (ConditionStepOutput, error) {
		return ConditionStepOutput{RandomNumber: rand.Intn(100) + 1}, nil //nolint:gosec
	},
		hatchet.WithParents(start),
		hatchet.WithWaitFor(hatchet.OrCondition(
			hatchet.SleepCondition(60*time.Second),
			hatchet.UserEventCondition("conditions-e2e:start", "true"),
		)),
	)

	workflow.NewTask("sum", func(ctx hatchet.Context, input any) (ConditionSumOutput, error) {
		var startOut, waitSleepOut, waitEventOut ConditionStepOutput
		_ = ctx.ParentOutput(start, &startOut)
		_ = ctx.ParentOutput(waitForSleep, &waitSleepOut)
		_ = ctx.ParentOutput(waitForEvent, &waitEventOut)

		sum := startOut.RandomNumber + waitSleepOut.RandomNumber + waitEventOut.RandomNumber

		var leftOut, rightOut ConditionStepOutput
		if err := ctx.ParentOutput(leftBranch, &leftOut); err == nil {
			sum += leftOut.RandomNumber
		}
		if err := ctx.ParentOutput(rightBranch, &rightOut); err == nil {
			sum += rightOut.RandomNumber
		}

		return ConditionSumOutput{Sum: sum}, nil
	}, hatchet.WithParents(start, waitForSleep, waitForEvent, skipOnEvent, leftBranch, rightBranch))

	worker, err := client.NewWorker("conditions-e2e-worker", hatchet.WithWorkflows(workflow))
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	ref, err := workflow.RunNoWait(bgCtx(), nil)
	require.NoError(t, err)

	time.Sleep(15 * time.Second)

	require.NoError(t, client.Events().Push(bgCtx(), "conditions-e2e:skip", map[string]any{}))
	require.NoError(t, client.Events().Push(bgCtx(), "conditions-e2e:start", map[string]any{}))

	result, err := ref.Result()
	require.NoError(t, err)

	var startOut ConditionStepOutput
	require.NoError(t, result.TaskOutput("start").Into(&startOut))

	var leftOut, rightOut map[string]any
	_ = result.TaskOutput("left-branch").Into(&leftOut)
	_ = result.TaskOutput("right-branch").Into(&rightOut)

	leftSkipped := leftOut["skipped"] == true
	rightSkipped := rightOut["skipped"] == true
	assert.True(t, leftSkipped || rightSkipped, "one branch should be skipped")

	var skipOnEventOut map[string]any
	_ = result.TaskOutput("skip-on-event").Into(&skipOnEventOut)
	assert.Equal(t, true, skipOnEventOut["skipped"])
}
