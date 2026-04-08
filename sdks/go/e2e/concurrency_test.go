//go:build e2e

package e2e

import (
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func TestConcurrencyCancelInProgress(t *testing.T) {
	client := newClient(t)

	type GroupInput struct {
		Group string `json:"group"`
	}

	var maxRuns int32 = 1
	strategy := types.CancelInProgress

	task := client.NewStandaloneTask("concurrency-cip-e2e", func(ctx hatchet.Context, input GroupInput) (any, error) {
		for i := 0; i < 50; i++ {
			select {
			case <-ctx.Done():
				return nil, nil
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
		return nil, nil
	},
		hatchet.WithWorkflowConcurrency(types.Concurrency{
			Expression:    "input.group",
			MaxRuns:       &maxRuns,
			LimitStrategy: &strategy,
		}),
	)

	worker, err := client.NewWorker("concurrency-cip-e2e-worker", hatchet.WithWorkflows(task))
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	testRunID := uuid.New().String()
	var refs []*hatchet.WorkflowRunRef

	for i := 0; i < 10; i++ {
		ref, err := task.RunNoWait(bgCtx(), GroupInput{Group: "A"},
			hatchet.WithRunMetadata(map[string]string{
				"test_run_id": testRunID,
				"i":           string(rune('0' + i)),
			}),
		)
		require.NoError(t, err)
		refs = append(refs, ref)
		time.Sleep(1 * time.Second)
	}

	for _, ref := range refs {
		_, _ = ref.Result()
	}

	time.Sleep(5 * time.Second)

	runs, err := client.Runs().List(bgCtx(), rest.V1WorkflowRunListParams{
		AdditionalMetadata: &[]string{"test_run_id:" + testRunID},
	})
	require.NoError(t, err)
	require.Len(t, runs.Rows, 10)

	sort.Slice(runs.Rows, func(i, j int) bool {
		return runs.Rows[i].CreatedAt.Before(runs.Rows[j].CreatedAt)
	})

	lastRun := runs.Rows[len(runs.Rows)-1]
	assert.Equal(t, rest.V1TaskStatusCOMPLETED, lastRun.Status)

	cancelledCount := 0
	for _, r := range runs.Rows[:len(runs.Rows)-1] {
		if r.Status == rest.V1TaskStatusCANCELLED {
			cancelledCount++
		}
	}
	assert.Equal(t, 9, cancelledCount)
}
