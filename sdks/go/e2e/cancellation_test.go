//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func TestCancellation(t *testing.T) {
	client := newClient(t)

	workflow := client.NewWorkflow("cancellation-e2e")

	workflow.NewTask("long-task", func(ctx hatchet.Context, input any) (any, error) {
		for i := 0; i < 30; i++ {
			select {
			case <-ctx.Done():
				return nil, nil
			default:
				time.Sleep(1 * time.Second)
			}
		}
		return map[string]string{"status": "completed"}, nil
	}, hatchet.WithExecutionTimeout(60*time.Second))

	worker, err := client.NewWorker("cancellation-e2e-worker", hatchet.WithWorkflows(workflow))
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	ref, err := workflow.RunNoWait(bgCtx(), nil)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)

	_, err = client.Runs().Cancel(bgCtx(), rest.V1CancelTaskRequest{
		ExternalIds: &[]types.UUID{uuid.MustParse(ref.RunId)},
	})
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	for i := 0; i < 30; i++ {
		status, err := client.Runs().GetStatus(bgCtx(), ref.RunId)
		require.NoError(t, err)

		if *status == rest.V1TaskStatusRUNNING {
			time.Sleep(1 * time.Second)
			continue
		}

		assert.Equal(t, rest.V1TaskStatusCANCELLED, *status)
		return
	}

	t.Fatal("workflow run did not cancel in time")
}
