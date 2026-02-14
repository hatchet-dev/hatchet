//go:build e2e

package e2e

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func TestOnFailure(t *testing.T) {
	client := newClient(t)

	workflow := client.NewWorkflow("on-failure-e2e")

	workflow.NewTask("failing-step", func(ctx hatchet.Context, input any) (any, error) {
		return nil, errors.New("step1 failed")
	}, hatchet.WithExecutionTimeout(5*time.Second))

	workflow.OnFailure(func(ctx hatchet.Context, input any) (map[string]string, error) {
		return map[string]string{"status": "handled"}, nil
	})

	worker, err := client.NewWorker("on-failure-e2e-worker", hatchet.WithWorkflows(workflow), hatchet.WithSlots(4))
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	ref, err := workflow.RunNoWait(bgCtx(), nil)
	require.NoError(t, err)

	_, err = ref.Result()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "step1 failed")

	time.Sleep(5 * time.Second)

	details, err := client.Runs().Get(bgCtx(), ref.RunId)
	require.NoError(t, err)

	require.Len(t, details.Tasks, 2)

	var completedCount, failedCount int
	for _, task := range details.Tasks {
		switch task.Status {
		case rest.V1TaskStatusCOMPLETED:
			completedCount++
			assert.Contains(t, task.DisplayName, "on_failure")
		case rest.V1TaskStatusFAILED:
			failedCount++
			assert.Contains(t, task.DisplayName, "failing-step")
		}
	}

	assert.Equal(t, 1, completedCount)
	assert.Equal(t, 1, failedCount)
}
