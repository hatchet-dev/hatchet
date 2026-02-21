//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type DAGStepOutput struct {
	Step   int `json:"step"`
	Result int `json:"result"`
}

func TestDAG(t *testing.T) {
	client := newClient(t)

	type Input struct {
		Value int `json:"value"`
	}

	workflow := client.NewWorkflow("dag-e2e")

	step1 := workflow.NewTask("step-1", func(ctx hatchet.Context, input Input) (DAGStepOutput, error) {
		return DAGStepOutput{Step: 1, Result: input.Value * 2}, nil
	})

	step2 := workflow.NewTask("step-2", func(ctx hatchet.Context, input Input) (DAGStepOutput, error) {
		var s1 DAGStepOutput
		if err := ctx.ParentOutput(step1, &s1); err != nil {
			return DAGStepOutput{}, err
		}
		return DAGStepOutput{Step: 2, Result: s1.Result + 10}, nil
	}, hatchet.WithParents(step1))

	step3 := workflow.NewTask("step-3", func(ctx hatchet.Context, input Input) (DAGStepOutput, error) {
		var s1 DAGStepOutput
		if err := ctx.ParentOutput(step1, &s1); err != nil {
			return DAGStepOutput{}, err
		}
		return DAGStepOutput{Step: 3, Result: s1.Result * 3}, nil
	}, hatchet.WithParents(step1))

	workflow.NewTask("final-step", func(ctx hatchet.Context, input Input) (DAGStepOutput, error) {
		var s2, s3 DAGStepOutput
		if err := ctx.ParentOutput(step2, &s2); err != nil {
			return DAGStepOutput{}, err
		}
		if err := ctx.ParentOutput(step3, &s3); err != nil {
			return DAGStepOutput{}, err
		}
		return DAGStepOutput{Step: 4, Result: s2.Result + s3.Result}, nil
	}, hatchet.WithParents(step2, step3))

	worker, err := client.NewWorker("dag-e2e-worker", hatchet.WithWorkflows(workflow))
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	result, err := workflow.Run(bgCtx(), Input{Value: 5})
	require.NoError(t, err)

	var s1, s2, s3, final DAGStepOutput
	require.NoError(t, result.TaskOutput("step-1").Into(&s1))
	require.NoError(t, result.TaskOutput("step-2").Into(&s2))
	require.NoError(t, result.TaskOutput("step-3").Into(&s3))
	require.NoError(t, result.TaskOutput("final-step").Into(&final))

	assert.Equal(t, 10, s1.Result)       // 5 * 2
	assert.Equal(t, 20, s2.Result)       // 10 + 10
	assert.Equal(t, 30, s3.Result)       // 10 * 3
	assert.Equal(t, 50, final.Result)    // 20 + 30
}
