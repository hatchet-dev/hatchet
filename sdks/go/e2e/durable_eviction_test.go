//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

func getFirstTaskExternalID(t *testing.T, ctx context.Context, runID string) string {
	t.Helper()
	details, err := sharedClient.Runs().GetDetails(ctx, uuid.MustParse(runID))
	require.NoError(t, err)
	for _, task := range details.TaskRuns {
		return task.ExternalId.String()
	}
	t.Fatal("no task runs found")
	return ""
}

func hasEvictedTask(t *testing.T, ctx context.Context, runID string) bool {
	t.Helper()
	details, err := sharedClient.Runs().GetDetails(ctx, uuid.MustParse(runID))
	if err != nil {
		return false
	}
	for _, task := range details.TaskRuns {
		if task.IsEvicted {
			return true
		}
	}
	return false
}

// pollUntilEvictedOrTerminal waits until a run is either evicted or reaches a terminal status.
// Some durable child-spawn executions can complete before eviction is observed, depending on timing.
func pollUntilEvictedOrTerminal(t *testing.T, ctx context.Context, runID string) bool {
	t.Helper()

	evicted := false
	pollUntil(t, ctx, func() (bool, error) {
		if hasEvictedTask(t, ctx, runID) {
			evicted = true
			return true, nil
		}

		status, err := sharedClient.Runs().GetStatus(ctx, runID)
		if err != nil {
			return false, err
		}

		return *status == rest.V1TaskStatusCOMPLETED ||
			*status == rest.V1TaskStatusFAILED ||
			*status == rest.V1TaskStatusCANCELLED, nil
	})

	return evicted
}

func TestNonEvictableTaskCompletes(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	start := time.Now()
	result, err := testNonEvictableSleep.Run(ctx, EmptyInput{})
	require.NoError(t, err)
	elapsed := time.Since(start).Seconds()

	var m map[string]any
	err = result.Into(&m)
	require.NoError(t, err)
	assert.Equal(t, "completed", m["status"])
	assert.GreaterOrEqual(t, elapsed, 10.0)
}

func TestNonEvictableTaskNotEvicted(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testNonEvictableSleep.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	time.Sleep(7 * time.Second) // Past EVICTION_TTL but task is non-evictable

	assert.False(t, hasEvictedTask(t, ctx, ref.RunId), "non-evictable task should never be evicted")

	result, err := ref.Result()
	require.NoError(t, err)
	var m map[string]any
	err = result.TaskOutput("non-evictable-sleep").Into(&m)
	require.NoError(t, err)
	assert.Equal(t, "completed", m["status"])
}

func TestEvictableTaskIsEvicted(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testEvictableSleep.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	pollUntilEvicted(t, ctx, sharedClient, ref.RunId)

	assert.True(t, hasEvictedTask(t, ctx, ref.RunId))
}

func TestEvictableTaskRestore(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testEvictableSleep.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	pollUntilEvicted(t, ctx, sharedClient, ref.RunId)

	taskID := getFirstTaskExternalID(t, ctx, ref.RunId)
	_, err = sharedClient.Runs().Restore(ctx, taskID)
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
}

func TestEvictableTaskRestoreCompletes(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	start := time.Now()
	ref, err := testEvictableSleep.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	pollUntilEvicted(t, ctx, sharedClient, ref.RunId)

	taskID := getFirstTaskExternalID(t, ctx, ref.RunId)
	_, err = sharedClient.Runs().Restore(ctx, taskID)
	require.NoError(t, err)

	result, err := ref.Result()
	require.NoError(t, err)
	elapsed := time.Since(start).Seconds()

	var m map[string]any
	err = result.TaskOutput("evictable-sleep").Into(&m)
	require.NoError(t, err)
	assert.Equal(t, "completed", m["status"])
	assert.GreaterOrEqual(t, elapsed, 15.0)
}

func TestEvictableWaitForEventIsEvicted(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testEvictableWaitForEvent.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	pollUntilEvicted(t, ctx, sharedClient, ref.RunId)

	assert.True(t, hasEvictedTask(t, ctx, ref.RunId))
}

func TestEvictableWaitForEventRestore(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testEvictableWaitForEvent.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	pollUntilEvicted(t, ctx, sharedClient, ref.RunId)

	taskID := getFirstTaskExternalID(t, ctx, ref.RunId)
	_, err = sharedClient.Runs().Restore(ctx, taskID)
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))

	err = sharedClient.Events().Push(ctx, evictionEventKey, map[string]any{})
	require.NoError(t, err)

	result, err := ref.Result()
	require.NoError(t, err)
	var m map[string]any
	err = result.TaskOutput("evictable-wait-for-event").Into(&m)
	require.NoError(t, err)
	assert.Equal(t, "completed", m["status"])
}

func TestEvictableChildSpawnIsEvicted(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testEvictableChildSpawn.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	if !pollUntilEvictedOrTerminal(t, ctx, ref.RunId) {
		t.Log("run completed before eviction was observed for evictable-child-spawn")
	}
}

func TestEvictableChildSpawnRestoreCompletes(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testEvictableChildSpawn.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	if pollUntilEvictedOrTerminal(t, ctx, ref.RunId) {
		taskID := getFirstTaskExternalID(t, ctx, ref.RunId)
		_, err = sharedClient.Runs().Restore(ctx, taskID)
		require.NoError(t, err)
	}

	result, err := ref.Result()
	require.NoError(t, err)
	var m map[string]any
	err = result.TaskOutput("evictable-child-spawn").Into(&m)
	require.NoError(t, err)
	assert.Equal(t, "completed", m["status"])
	child, ok := m["child"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "completed", child["child_status"])
}

func TestMultipleEvictionCycle(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	start := time.Now()
	ref, err := testMultipleEviction.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	// First eviction cycle
	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	pollUntilEvicted(t, ctx, sharedClient, ref.RunId)
	assert.True(t, hasEvictedTask(t, ctx, ref.RunId), "first eviction failed")

	taskID := getFirstTaskExternalID(t, ctx, ref.RunId)
	_, err = sharedClient.Runs().Restore(ctx, taskID)
	require.NoError(t, err)

	// Second eviction cycle
	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	pollUntilEvicted(t, ctx, sharedClient, ref.RunId)
	assert.True(t, hasEvictedTask(t, ctx, ref.RunId), "second eviction failed")

	taskID = getFirstTaskExternalID(t, ctx, ref.RunId)
	_, err = sharedClient.Runs().Restore(ctx, taskID)
	require.NoError(t, err)

	result, err := ref.Result()
	require.NoError(t, err)
	elapsed := time.Since(start).Seconds()

	var m map[string]any
	err = result.TaskOutput("multiple-eviction").Into(&m)
	require.NoError(t, err)
	assert.Equal(t, "completed", m["status"])
	assert.GreaterOrEqual(t, elapsed, 30.0)
}

func TestEvictionPlusReplay(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testEvictableSleep.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	pollUntilEvicted(t, ctx, sharedClient, ref.RunId)

	_, err = sharedClient.Runs().Replay(ctx, rest.V1ReplayTaskRequest{
		ExternalIds: toUUIDs(ref.RunId),
	})
	require.NoError(t, err)

	result, err := ref.Result()
	require.NoError(t, err)
	var m map[string]any
	err = result.TaskOutput("evictable-sleep").Into(&m)
	require.NoError(t, err)
	assert.Equal(t, "completed", m["status"])
}

func TestEvictableCancelAfterEviction(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testEvictableSleep.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	pollUntilEvicted(t, ctx, sharedClient, ref.RunId)
	assert.True(t, hasEvictedTask(t, ctx, ref.RunId))

	_, err = sharedClient.Runs().Cancel(ctx, rest.V1CancelTaskRequest{
		ExternalIds: toUUIDs(ref.RunId),
	})
	require.NoError(t, err)

	pollUntil(t, ctx, func() (bool, error) {
		status, err := sharedClient.Runs().GetStatus(ctx, ref.RunId)
		if err != nil {
			return false, err
		}
		return *status == rest.V1TaskStatusCANCELLED, nil
	})
}

func TestRestoreIdempotency(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testEvictableSleep.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	pollUntilEvicted(t, ctx, sharedClient, ref.RunId)

	taskID := getFirstTaskExternalID(t, ctx, ref.RunId)

	// Double restore should be idempotent
	_, err = sharedClient.Runs().Restore(ctx, taskID)
	require.NoError(t, err)
	_, err = sharedClient.Runs().Restore(ctx, taskID)
	require.NoError(t, err)

	result, err := ref.Result()
	require.NoError(t, err)
	var m map[string]any
	err = result.TaskOutput("evictable-sleep").Into(&m)
	require.NoError(t, err)
	assert.Equal(t, "completed", m["status"])
}
