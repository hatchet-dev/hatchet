//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func toUUIDs(ids ...string) *[]openapi_types.UUID {
	uuids := make([]openapi_types.UUID, len(ids))
	for i, id := range ids {
		uuids[i] = uuid.MustParse(id)
	}
	return &uuids
}

// taskExternalID extracts the first task external ID from run details.
func taskExternalID(t *testing.T, client *hatchet.Client, ctx context.Context, runID string) string {
	t.Helper()
	details, err := client.Runs().GetDetails(ctx, uuid.MustParse(runID))
	require.NoError(t, err)
	for _, task := range details.TaskRuns {
		return task.ExternalId.String()
	}
	t.Fatal("no task runs found")
	return ""
}

func TestDurableWorkflow(t *testing.T) {
	ctx := newTestContext(t)

	ref, err := testDurableWorkflow.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	id := uniqueID()

	time.Sleep(time.Duration(sleepTime+10) * time.Second)

	err = sharedClient.Events().Push(ctx, eventKey, AwaitedEvent{ID: id})
	require.NoError(t, err)

	result, err := ref.Result()
	require.NoError(t, err)

	durableOutput := resultMap(t, result, "durable_task")
	assert.Equal(t, "success", durableOutput["status"])
	assert.Equal(t, float64(sleepTime), durableOutput["sleep_duration_seconds"])

	workers, err := sharedClient.Workers().List(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, workers.Rows)
}

func TestDurableSleepCancelReplay(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testWaitForSleepTwice.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	time.Sleep(time.Duration(sleepTime/2) * time.Second)

	_, err = sharedClient.Runs().Cancel(ctx, rest.V1CancelTaskRequest{
		ExternalIds: toUUIDs(ref.RunId),
	})
	require.NoError(t, err)

	// Wait for cancellation
	time.Sleep(2 * time.Second)

	replayStart := time.Now()
	_, err = sharedClient.Runs().Replay(ctx, rest.V1ReplayTaskRequest{
		ExternalIds: toUUIDs(ref.RunId),
	})
	require.NoError(t, err)

	result, err := ref.Result()
	require.NoError(t, err)
	replayElapsed := time.Since(replayStart).Seconds()

	var output map[string]float64
	err = result.TaskOutput("wait-for-sleep-twice").Into(&output)
	require.NoError(t, err)

	assert.Less(t, output["runtime"], float64(sleepTime))
	assert.LessOrEqual(t, replayElapsed, float64(sleepTime))
}

func TestDurableChildSpawn(t *testing.T) {
	ctx := newTestContext(t)

	result, err := testDurableWithSpawn.Run(ctx, EmptyInput{})
	require.NoError(t, err)

	var m map[string]any
	err = result.Into(&m)
	require.NoError(t, err)
	childOutput, ok := m["child_output"].(map[string]any)
	require.True(t, ok, "expected child_output to be a map")
	assert.Equal(t, "hello from child 1", childOutput["message"])
}

func TestDurableChildBulkSpawn(t *testing.T) {
	ctx := newTestContext(t)
	n := 8

	result, err := testDurableWithBulkSpawn.Run(ctx, DurableBulkSpawnInput{N: n})
	require.NoError(t, err)

	var m map[string]any
	err = result.Into(&m)
	require.NoError(t, err)
	outputs, ok := m["child_outputs"].([]any)
	require.True(t, ok, "expected child_outputs to be an array")
	assert.GreaterOrEqual(t, len(outputs), n-1)
	assert.LessOrEqual(t, len(outputs), n)

	seen := make(map[string]struct{}, len(outputs))
	for _, raw := range outputs {
		child, ok := raw.(map[string]any)
		require.True(t, ok, "expected each child output to be an object")

		msg, ok := child["message"].(string)
		require.True(t, ok, "expected child message to be a string")
		seen[msg] = struct{}{}
	}
	assert.Len(t, seen, len(outputs))
}

func TestDurableSleepEventSpawnReplay(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	start := time.Now()
	ref, err := testDurableSleepEventSpawn.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	time.Sleep(time.Duration(sleepTime+5) * time.Second)
	err = sharedClient.Events().Push(ctx, eventKey, map[string]string{"test": "test"})
	require.NoError(t, err)

	result, err := ref.Result()
	require.NoError(t, err)
	firstElapsed := time.Since(start).Seconds()

	m := resultMap(t, result, "durable-sleep-event-spawn")
	childOutput, ok := m["child_output"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "hello from child 1", childOutput["message"])
	assert.GreaterOrEqual(t, firstElapsed, float64(sleepTime))

	replayStart := time.Now()
	_, err = sharedClient.Runs().Replay(ctx, rest.V1ReplayTaskRequest{
		ExternalIds: toUUIDs(ref.RunId),
	})
	require.NoError(t, err)

	replayResult, err := ref.Result()
	require.NoError(t, err)
	replayElapsed := time.Since(replayStart).Seconds()

	rm := resultMap(t, replayResult, "durable-sleep-event-spawn")
	replayChild, ok := rm["child_output"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "hello from child 1", replayChild["message"])
	assert.Less(t, replayElapsed, float64(sleepTime))
}

func TestDurableCompletedReplay(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testWaitForSleepTwice.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	start := time.Now()
	result, err := ref.Result()
	require.NoError(t, err)
	elapsed := time.Since(start).Seconds()

	var firstOutput map[string]float64
	err = result.TaskOutput("wait-for-sleep-twice").Into(&firstOutput)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, firstOutput["runtime"], float64(sleepTime))
	assert.GreaterOrEqual(t, elapsed, float64(sleepTime))

	start = time.Now()
	_, err = sharedClient.Runs().Replay(ctx, rest.V1ReplayTaskRequest{
		ExternalIds: toUUIDs(ref.RunId),
	})
	require.NoError(t, err)

	replayResult, err := ref.Result()
	require.NoError(t, err)
	elapsed = time.Since(start).Seconds()

	var replayOutput map[string]float64
	err = replayResult.TaskOutput("wait-for-sleep-twice").Into(&replayOutput)
	require.NoError(t, err)
	assert.Less(t, replayOutput["runtime"], float64(sleepTime))
	assert.Less(t, elapsed, float64(sleepTime))
}

func TestDurableSpawnDAG(t *testing.T) {
	ctx := newTestContext(t)

	start := time.Now()
	result, err := testDurableSpawnDAG.Run(ctx, EmptyInput{})
	require.NoError(t, err)
	elapsed := time.Since(start).Seconds()

	var m map[string]any
	err = result.Into(&m)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, m["sleep_duration"].(float64), 1.0)
	assert.GreaterOrEqual(t, m["spawn_duration"].(float64), 5.0)
	assert.GreaterOrEqual(t, elapsed, 5.0)
	assert.LessOrEqual(t, elapsed, 15.0)
}

func TestDurableNonDeterminism(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testDurableNonDeterminism.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	result, err := ref.Result()
	require.NoError(t, err)

	var output NonDeterminismOutput
	err = result.TaskOutput("durable-non-determinism").Into(&output)
	require.NoError(t, err)

	assert.Greater(t, output.SleepTime, output.AttemptNumber)
	assert.Less(t, output.SleepTime, output.AttemptNumber*3)
	assert.False(t, output.NonDeterminismDetected)

	_, err = sharedClient.Runs().Replay(ctx, rest.V1ReplayTaskRequest{
		ExternalIds: toUUIDs(ref.RunId),
	})
	require.NoError(t, err)

	replayResult, err := ref.Result()
	if err != nil {
		assert.Contains(t, err.Error(), "non-determinism error")
		return
	}

	var replayOutput NonDeterminismOutput
	err = replayResult.TaskOutput("durable-non-determinism").Into(&replayOutput)
	require.NoError(t, err)

	assert.True(t, replayOutput.NonDeterminismDetected)
	assert.NotNil(t, replayOutput.NodeID)
	if replayOutput.NodeID != nil {
		assert.Equal(t, 1, *replayOutput.NodeID)
	}
	assert.Equal(t, 2, replayOutput.AttemptNumber)
}

func TestDurableReplayReset(t *testing.T) {
	requireDurableEviction(t)

	for _, nodeID := range []int64{1, 2, 3} {
		t.Run("node_"+string(rune('0'+nodeID)), func(t *testing.T) {
			ctx := newTestContext(t)

			ref, err := testDurableReplayReset.RunNoWait(ctx, EmptyInput{})
			require.NoError(t, err)

			result, err := ref.Result()
			require.NoError(t, err)

			var output ReplayResetResponse
			err = result.TaskOutput("durable-replay-reset").Into(&output)
			require.NoError(t, err)

			assert.GreaterOrEqual(t, output.Sleep1Duration, float64(replayResetSleepTime))
			assert.GreaterOrEqual(t, output.Sleep2Duration, float64(replayResetSleepTime))
			assert.GreaterOrEqual(t, output.Sleep3Duration, float64(replayResetSleepTime))

			_, err = sharedClient.Runs().BranchDurableTask(ctx, ref.RunId, nodeID, 1)
			require.NoError(t, err)

			start := time.Now()
			pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
			resetResult, err := ref.Result()
			require.NoError(t, err)
			resetElapsed := time.Since(start).Seconds()

			var resetOutput ReplayResetResponse
			err = resetResult.TaskOutput("durable-replay-reset").Into(&resetOutput)
			require.NoError(t, err)

			durations := []float64{resetOutput.Sleep1Duration, resetOutput.Sleep2Duration, resetOutput.Sleep3Duration}
			for i, d := range durations {
				if int64(i+1) < nodeID {
					assert.Less(t, d, float64(replayResetSleepTime))
				} else {
					assert.GreaterOrEqual(t, d, float64(replayResetSleepTime))
				}
			}

			sleepsToRedo := 3 - int(nodeID) + 1
			assert.GreaterOrEqual(t, resetElapsed, float64(sleepsToRedo*replayResetSleepTime))
		})
	}
}

func TestDurableBranchingOffBranch(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testDurableReplayReset.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	result, err := ref.Result()
	require.NoError(t, err)

	var output ReplayResetResponse
	err = result.TaskOutput("durable-replay-reset").Into(&output)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, output.Sleep1Duration, float64(replayResetSleepTime))
	assert.GreaterOrEqual(t, output.Sleep2Duration, float64(replayResetSleepTime))
	assert.GreaterOrEqual(t, output.Sleep3Duration, float64(replayResetSleepTime))

	// Branch 1: reset from node 1
	_, err = sharedClient.Runs().BranchDurableTask(ctx, ref.RunId, 1, 1)
	require.NoError(t, err)

	start := time.Now()
	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	resetResult, err := ref.Result()
	require.NoError(t, err)
	resetElapsed := time.Since(start).Seconds()

	var resetOutput ReplayResetResponse
	err = resetResult.TaskOutput("durable-replay-reset").Into(&resetOutput)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, resetOutput.Sleep1Duration, float64(replayResetSleepTime))
	assert.GreaterOrEqual(t, resetOutput.Sleep2Duration, float64(replayResetSleepTime))
	assert.GreaterOrEqual(t, resetOutput.Sleep3Duration, float64(replayResetSleepTime))
	assert.GreaterOrEqual(t, resetElapsed, float64(3*replayResetSleepTime))

	// Branch 2: reset from node 2, branching off branch 2
	_, err = sharedClient.Runs().BranchDurableTask(ctx, ref.RunId, 2, 2)
	require.NoError(t, err)

	start = time.Now()
	pollUntilRunStatus(t, ctx, sharedClient, ref.RunId, string(rest.V1TaskStatusRUNNING))
	resetResult2, err := ref.Result()
	require.NoError(t, err)
	resetElapsed2 := time.Since(start).Seconds()

	var resetOutput2 ReplayResetResponse
	err = resetResult2.TaskOutput("durable-replay-reset").Into(&resetOutput2)
	require.NoError(t, err)

	assert.Less(t, resetOutput2.Sleep1Duration, float64(replayResetSleepTime))
	assert.GreaterOrEqual(t, resetOutput2.Sleep2Duration, float64(replayResetSleepTime))
	assert.GreaterOrEqual(t, resetOutput2.Sleep3Duration, float64(replayResetSleepTime))
	assert.GreaterOrEqual(t, resetElapsed2, float64(2*replayResetSleepTime))
}

func TestDurableMemoizationViaReplay(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	message := uniqueID()
	start := time.Now()
	ref, err := testMemoTask.RunNoWait(ctx, MemoInput{Message: message})
	require.NoError(t, err)

	result1, err := ref.Result()
	require.NoError(t, err)
	duration1 := time.Since(start).Seconds()

	var output1 SleepResult
	err = result1.TaskOutput("memo-task").Into(&output1)
	require.NoError(t, err)

	_, err = sharedClient.Runs().Replay(ctx, rest.V1ReplayTaskRequest{
		ExternalIds: toUUIDs(ref.RunId),
	})
	require.NoError(t, err)

	start = time.Now()
	result2, err := ref.Result()
	require.NoError(t, err)
	duration2 := time.Since(start).Seconds()

	var output2 SleepResult
	err = result2.TaskOutput("memo-task").Into(&output2)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, duration1, float64(sleepTime))
	assert.Less(t, duration2, 1.0)
	assert.Equal(t, output1.Message, output2.Message)
}

func TestDurableMemoNowCaching(t *testing.T) {
	requireDurableEviction(t)
	ctx := newTestContext(t)

	ref, err := testMemoNowCaching.RunNoWait(ctx, EmptyInput{})
	require.NoError(t, err)

	result1, err := ref.Result()
	require.NoError(t, err)

	m1 := resultMap(t, result1, "memo-now-caching")

	_, err = sharedClient.Runs().Replay(ctx, rest.V1ReplayTaskRequest{
		ExternalIds: toUUIDs(ref.RunId),
	})
	require.NoError(t, err)

	result2, err := ref.Result()
	require.NoError(t, err)

	m2 := resultMap(t, result2, "memo-now-caching")

	assert.Equal(t, m1["start_time"], m2["start_time"])
}
