//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func TestExecutionTimeout(t *testing.T) {
	client := newClient(t)

	workflow := client.NewWorkflow("timeout-e2e")

	workflow.NewTask("timeout-task", func(ctx hatchet.Context, input any) (any, error) {
		time.Sleep(30 * time.Second)
		return map[string]string{"status": "success"}, nil
	}, hatchet.WithExecutionTimeout(5*time.Second))

	worker, err := client.NewWorker("timeout-e2e-worker", hatchet.WithWorkflows(workflow), hatchet.WithSlots(4))
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	ref, err := workflow.RunNoWait(bgCtx(), nil)
	require.NoError(t, err)

	_, err = ref.Result()
	require.Error(t, err)
	assert.Regexp(t, "(?i)(timeout|TIMED_OUT|failed)", err.Error())
}

type RefreshOutput struct {
	Status string `json:"status"`
}

func TestRefreshTimeout(t *testing.T) {
	client := newClient(t)

	workflow := client.NewWorkflow("refresh-timeout-e2e")

	workflow.NewTask("refresh-task", func(ctx hatchet.Context, input any) (RefreshOutput, error) {
		err := ctx.RefreshTimeout("10s")
		if err != nil {
			return RefreshOutput{}, err
		}
		time.Sleep(5 * time.Second)
		return RefreshOutput{Status: "success"}, nil
	}, hatchet.WithExecutionTimeout(3*time.Second))

	worker, err := client.NewWorker("refresh-timeout-e2e-worker", hatchet.WithWorkflows(workflow), hatchet.WithSlots(4))
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	result, err := workflow.Run(bgCtx(), nil)
	require.NoError(t, err)

	var out RefreshOutput
	require.NoError(t, result.TaskOutput("refresh-task").Into(&out))
	assert.Equal(t, "success", out.Status)
}
