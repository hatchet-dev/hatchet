//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// TestBatchFlushesWhenSizeReached mirrors Python's test_flushes_when_batch_size_is_reached.
func TestBatchFlushesWhenSizeReached(t *testing.T) {
	ctx := newTestContext(t)

	inputs := []string{"alpha", "bravo", "charlie"}
	results := make([]SimpleOutput, len(inputs))

	g, gctx := errgroup.WithContext(ctx)
	for i, msg := range inputs {
		i, msg := i, msg
		g.Go(func() error {
			result, err := testBatchSimple.Run(gctx, SimpleInput{Message: msg})
			if err != nil {
				return err
			}
			return result.Into(&results[i])
		})
	}
	require.NoError(t, g.Wait())

	for i, msg := range inputs {
		assert.Equal(t, strings.ToUpper(msg), results[i].TransformedMessage)
	}
}

// TestBatchFlushesOnIntervalWithFewerItems mirrors Python's
// test_flushes_when_fewer_items_buffered_than_batch_size.
func TestBatchFlushesOnIntervalWithFewerItems(t *testing.T) {
	ctx := newTestContext(t)

	inputs := []string{"delta", "echo"}
	refs := make([]*hatchet.WorkflowRunRef, len(inputs))

	for i, msg := range inputs {
		ref, err := testBatchSimple.RunNoWait(ctx, SimpleInput{Message: msg})
		require.NoError(t, err)
		refs[i] = ref
	}

	time.Sleep(500 * time.Millisecond)

	for i, msg := range inputs {
		result, err := refs[i].Result()
		require.NoError(t, err)

		var output SimpleOutput
		require.NoError(t, result.TaskOutput("batch-simple").Into(&output))
		assert.Equal(t, strings.ToUpper(msg), output.TransformedMessage)
	}
}

// TestBatchPartitionsByKey mirrors Python's test_partitions_batches_by_key_when_batch_size_reached.
func TestBatchPartitionsByKey(t *testing.T) {
	ctx := newTestContext(t)

	inputs := []KeyedInput{
		{Message: "alpha", Group: "tenant-1"},
		{Message: "bravo", Group: "tenant-1"},
		{Message: "charlie", Group: "tenant-2"},
		{Message: "delta", Group: "tenant-2"},
	}

	results := make([]KeyedOutput, len(inputs))

	g, gctx := errgroup.WithContext(ctx)
	for i, inp := range inputs {
		i, inp := i, inp
		g.Go(func() error {
			result, err := testBatchKeyed.Run(gctx, inp)
			if err != nil {
				return err
			}
			return result.Into(&results[i])
		})
	}
	require.NoError(t, g.Wait())

	for i, inp := range inputs {
		require.NotNil(t, results[i].BatchKey)
		assert.Equal(t, inp.Group, *results[i].BatchKey)
		require.NotNil(t, results[i].BatchSize)
		assert.Equal(t, 2, *results[i].BatchSize)
		require.NotNil(t, results[i].UniqueKeys)
		assert.Equal(t, 1, *results[i].UniqueKeys)
		assert.Equal(t, strings.ToUpper(inp.Message), results[i].Uppercase)
	}
}

// TestBatchGroupKeyParseFailureIsolated mirrors Python's
// test_batch_group_key_parse_failure_fails_only_that_task.
func TestBatchGroupKeyParseFailureIsolated(t *testing.T) {
	ctx := newTestContext(t)

	goodRef, err := testBatchKeyedFailable.RunNoWait(ctx, KeyedFailableInput{Message: "hello", Group: "tenant-1"})
	require.NoError(t, err)

	badRef, err := testBatchKeyedFailable.RunNoWait(ctx, KeyedFailableInput{Message: "world", Group: 123})
	require.NoError(t, err)

	_, badErr := badRef.Result()
	require.Error(t, badErr)
	assert.Contains(t, badErr.Error(), "failed to parse batch group key expression")

	goodResult, err := goodRef.Result()
	require.NoError(t, err)

	var output KeyedOutput
	require.NoError(t, goodResult.TaskOutput("batch-keyed-failable").Into(&output))
	assert.Equal(t, "HELLO", output.Uppercase)
}

// TestBatchKeyedIntervalFlush mirrors Python's
// test_flushes_keyed_batches_independently_when_interval_elapses.
func TestBatchKeyedIntervalFlush(t *testing.T) {
	ctx := newTestContext(t)

	inputs := []KeyedInput{
		{Message: "echo", Group: "tenant-1"},
		{Message: "foxtrot", Group: "tenant-1"},
		{Message: "golf", Group: "tenant-1"},
		{Message: "hotel", Group: "tenant-2"},
	}

	results := make([]KeyedOutput, len(inputs))

	g, gctx := errgroup.WithContext(ctx)
	for i, inp := range inputs {
		i, inp := i, inp
		g.Go(func() error {
			result, err := testBatchKeyedInterval.Run(gctx, inp)
			if err != nil {
				return err
			}
			return result.Into(&results[i])
		})
	}
	require.NoError(t, g.Wait())

	for i, inp := range inputs {
		require.NotNil(t, results[i].BatchKey)
		assert.Equal(t, inp.Group, *results[i].BatchKey)
		require.NotNil(t, results[i].UniqueKeys)
		assert.Equal(t, 1, *results[i].UniqueKeys)
	}

	for i := 0; i < 3; i++ {
		require.NotNil(t, results[i].BatchSize)
		assert.Equal(t, 3, *results[i].BatchSize)
	}

	require.NotNil(t, results[3].BatchSize)
	assert.Equal(t, 1, *results[3].BatchSize)
	assert.Equal(t, "HOTEL", results[3].Uppercase)
}

// TestBatchLargePayloadMemoryFlush mirrors Python's test_completes_all_tasks_with_large_payloads.
func TestBatchLargePayloadMemoryFlush(t *testing.T) {
	ctx := newTestContext(t)

	const payloadSize = 100_000
	const taskCount = 100
	payload := strings.Repeat("x", payloadSize)

	results := make([]LargeOutput, taskCount)

	g, gctx := errgroup.WithContext(ctx)
	for i := 0; i < taskCount; i++ {
		i := i
		g.Go(func() error {
			result, err := testBatchLarge.Run(gctx, LargePayloadInput{Data: payload})
			if err != nil {
				return err
			}
			return result.Into(&results[i])
		})
	}
	require.NoError(t, g.Wait())

	batchIDs := make(map[string]struct{})
	for _, r := range results {
		batchIDs[r.BatchId] = struct{}{}
		assert.True(t, r.Received)
		assert.Equal(t, payloadSize, r.DataLength)
	}

	// The batch should have flushed 3 times due to the 4mb memory-size limit, even
	// though batch_max_size (100) was never reached by count alone.
	assert.Len(t, batchIDs, 3)
}

// TestBatchSizeOneNoKeys mirrors Python's test_handles_batch_size_of_one_without_keys.
func TestBatchSizeOneNoKeys(t *testing.T) {
	ctx := newTestContext(t)

	inputs := []string{"india", "juliet"}
	results := make([]SingleOutput, len(inputs))

	g, gctx := errgroup.WithContext(ctx)
	for i, msg := range inputs {
		i, msg := i, msg
		g.Go(func() error {
			result, err := testBatchSingle.Run(gctx, SimpleInput{Message: msg})
			if err != nil {
				return err
			}
			return result.Into(&results[i])
		})
	}
	require.NoError(t, g.Wait())

	for i, msg := range inputs {
		assert.Equal(t, 1, results[i].BatchSize)
		assert.Equal(t, msg, results[i].Original)
	}
}

// TestBatchResultsPreserveSubmissionOrder mirrors Python's test_returns_results_in_submission_order.
func TestBatchResultsPreserveSubmissionOrder(t *testing.T) {
	ctx := newTestContext(t)

	const count = 20
	results := make([]OrderedOutput, count)

	g, gctx := errgroup.WithContext(ctx)
	for i := 0; i < count; i++ {
		i := i
		g.Go(func() error {
			result, err := testBatchOrdered.Run(gctx, OrderedInput{Index: i})
			if err != nil {
				return err
			}
			return result.Into(&results[i])
		})
	}
	require.NoError(t, g.Wait())

	for i := 0; i < count; i++ {
		assert.Equal(t, i, results[i].Index)
	}
}

// TestBatchBroadcastReturn mirrors Python's test_broadcasted_return.
func TestBatchBroadcastReturn(t *testing.T) {
	ctx := newTestContext(t)

	const count = 10
	results := make([]BroadcastOutput, count)

	g, gctx := errgroup.WithContext(ctx)
	for i := 0; i < count; i++ {
		i := i
		g.Go(func() error {
			result, err := testBatchBroadcast.Run(gctx, SimpleInput{Message: "hello"})
			if err != nil {
				return err
			}
			return result.Into(&results[i])
		})
	}
	require.NoError(t, g.Wait())

	for _, r := range results {
		assert.Equal(t, 50, r.Sum)
	}
}

// TestBatchChildSpawning mirrors Python's test_child_spawning.
func TestBatchChildSpawning(t *testing.T) {
	ctx := newTestContext(t)

	const count = 10
	// Each concurrent .Run() call is one batch member; a batch task's per-member result
	// (not the whole batch map) is what a caller's own .Run() resolves to.
	results := make([]ChildOutput, count)

	g, gctx := errgroup.WithContext(ctx)
	for i := 0; i < count; i++ {
		i := i
		g.Go(func() error {
			result, err := testBatchChildSpawn.Run(gctx, SimpleInput{Message: "hello"})
			if err != nil {
				return err
			}
			return result.Into(&results[i])
		})
	}
	require.NoError(t, g.Wait())

	for _, r := range results {
		assert.Equal(t, len("blahblah"), r.MessageLen)
	}
}

// TestBatchChildBatchSpawning mirrors Python's test_child_batch_spawning.
func TestBatchChildBatchSpawning(t *testing.T) {
	ctx := newTestContext(t)

	const count = 10
	// Each concurrent .Run() call is one batch member; a batch task's per-member result
	// (not the whole batch map) is what a caller's own .Run() resolves to.
	results := make([]ChildBatchOutput, count)

	g, gctx := errgroup.WithContext(ctx)
	for i := 0; i < count; i++ {
		i := i
		g.Go(func() error {
			result, err := testBatchChildBatchSpawn.Run(gctx, SimpleInput{Message: "hello"})
			if err != nil {
				return err
			}
			return result.Into(&results[i])
		})
	}
	require.NoError(t, g.Wait())

	for _, r := range results {
		require.NotEmpty(t, r.Out)
	}
}
