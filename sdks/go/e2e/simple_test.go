//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type SimpleOutput struct {
	Result string `json:"result"`
}

func TestSimpleTask_Run(t *testing.T) {
	client := newClient(t)

	task := client.NewStandaloneTask("simple-run", func(ctx hatchet.Context, input any) (SimpleOutput, error) {
		return SimpleOutput{Result: "Hello, world!"}, nil
	})

	worker, err := client.NewWorker("simple-run-worker", hatchet.WithWorkflows(task))
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	result, err := task.Run(bgCtx(), nil)
	require.NoError(t, err)

	var out SimpleOutput
	require.NoError(t, result.Into(&out))
	assert.Equal(t, "Hello, world!", out.Result)
}

func TestSimpleTask_RunNoWait(t *testing.T) {
	client := newClient(t)

	task := client.NewStandaloneTask("simple-no-wait", func(ctx hatchet.Context, input any) (SimpleOutput, error) {
		return SimpleOutput{Result: "Hello, world!"}, nil
	})

	worker, err := client.NewWorker("simple-no-wait-worker", hatchet.WithWorkflows(task))
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	ref, err := task.RunNoWait(bgCtx(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, ref.RunId)

	result, err := ref.Result()
	require.NoError(t, err)

	var out SimpleOutput
	require.NoError(t, result.TaskOutput("simple-no-wait").Into(&out))
	assert.Equal(t, "Hello, world!", out.Result)
}

func TestSimpleTask_RunMany(t *testing.T) {
	client := newClient(t)

	task := client.NewStandaloneTask("simple-run-many", func(ctx hatchet.Context, input any) (SimpleOutput, error) {
		return SimpleOutput{Result: "Hello, world!"}, nil
	})

	worker, err := client.NewWorker("simple-run-many-worker", hatchet.WithWorkflows(task))
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	refs, err := task.RunMany(bgCtx(), []hatchet.RunManyOpt{
		{Input: nil},
		{Input: nil},
		{Input: nil},
	})
	require.NoError(t, err)
	assert.Len(t, refs, 3)

	for _, ref := range refs {
		result, err := ref.Result()
		require.NoError(t, err)

		var out SimpleOutput
		require.NoError(t, result.TaskOutput("simple-run-many").Into(&out))
		assert.Equal(t, "Hello, world!", out.Result)
	}
}

func TestSimpleDurableTask_Run(t *testing.T) {
	client := newClient(t)

	task := client.NewStandaloneDurableTask("simple-durable-run", func(ctx hatchet.DurableContext, input any) (SimpleOutput, error) {
		return SimpleOutput{Result: "Hello, world!"}, nil
	})

	worker, err := client.NewWorker("simple-durable-worker", hatchet.WithWorkflows(task))
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	result, err := task.Run(bgCtx(), nil)
	require.NoError(t, err)

	var out SimpleOutput
	require.NoError(t, result.Into(&out))
	assert.Equal(t, "Hello, world!", out.Result)
}
