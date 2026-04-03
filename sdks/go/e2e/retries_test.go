//go:build e2e

package e2e

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/worker"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func TestRetriesWithCount(t *testing.T) {
	client := newClient(t)

	type RetryOutput struct {
		Message string `json:"message"`
	}

	task := client.NewStandaloneTask("retries-count-e2e", func(ctx hatchet.Context, input any) (RetryOutput, error) {
		if ctx.RetryCount() < 2 {
			return RetryOutput{}, errors.New("intentional failure")
		}
		return RetryOutput{Message: "success"}, nil
	}, hatchet.WithRetries(3))

	w, err := client.NewWorker("retries-count-e2e-worker", hatchet.WithWorkflows(task))
	require.NoError(t, err)
	cleanup := startWorker(t, w)
	defer cleanup() //nolint:errcheck

	result, err := task.Run(bgCtx(), nil)
	require.NoError(t, err)

	var out RetryOutput
	require.NoError(t, result.Into(&out))
	assert.Equal(t, "success", out.Message)
}

func TestNonRetryableError(t *testing.T) {
	client := newClient(t)

	task := client.NewStandaloneTask("non-retryable-e2e", func(ctx hatchet.Context, input any) (any, error) {
		return nil, worker.NewNonRetryableError(errors.New("non-retryable failure"))
	}, hatchet.WithRetries(3))

	w, err := client.NewWorker("non-retryable-e2e-worker", hatchet.WithWorkflows(task))
	require.NoError(t, err)
	cleanup := startWorker(t, w)
	defer cleanup() //nolint:errcheck

	_, err = task.Run(bgCtx(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-retryable failure")
}

func TestRetriesAlwaysFail(t *testing.T) {
	client := newClient(t)

	task := client.NewStandaloneTask("retries-always-fail-e2e", func(ctx hatchet.Context, input any) (any, error) {
		return nil, errors.New("always fails")
	}, hatchet.WithRetries(2))

	w, err := client.NewWorker("retries-always-fail-e2e-worker", hatchet.WithWorkflows(task))
	require.NoError(t, err)
	cleanup := startWorker(t, w)
	defer cleanup() //nolint:errcheck

	_, err = task.Run(bgCtx(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "always fails")
}
