//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// runConcurrent submits each input via runFn in a goroutine, collecting results in order.
func runConcurrent[I any, O any](t *testing.T, runFn func(ctx context.Context, inp I) (O, error), inputs []I) []O {
	t.Helper()
	ctx := newTestContext(t)
	results := make([]O, len(inputs))
	errs := make([]error, len(inputs))
	var wg sync.WaitGroup

	for i, inp := range inputs {
		wg.Add(1)
		go func(idx int, item I) {
			defer wg.Done()
			out, err := runFn(ctx, item)
			errs[idx] = err
			results[idx] = out
		}(i, inp)
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
	return results
}

func taskResultMap(t *testing.T, r *hatchet.TaskResult) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, r.Into(&m))
	return m
}

// TestBatchFlushOnSize verifies that a batch flushes when the max size is reached.
func TestBatchFlushOnSize(t *testing.T) {
	inputs := []string{"alpha", "bravo", "charlie"}

	results := runConcurrent(t, func(ctx context.Context, msg string) (map[string]any, error) {
		r, err := testBatchSimple.Run(ctx, SimpleInput{Message: msg})
		if err != nil {
			return nil, err
		}
		return taskResultMap(t, r), nil
	}, inputs)

	require.Len(t, results, 3)
	for i, result := range results {
		assert.Equal(t, strings.ToUpper(inputs[i]), result["TransformedMessage"])
	}
}

// TestBatchFlushOnInterval verifies that a batch flushes even when fewer items than max size
// are buffered, once the flush interval elapses.
func TestBatchFlushOnInterval(t *testing.T) {
	inputs := []string{"delta", "echo"}

	results := runConcurrent(t, func(ctx context.Context, msg string) (map[string]any, error) {
		r, err := testBatchSimple.Run(ctx, SimpleInput{Message: msg})
		if err != nil {
			return nil, err
		}
		return taskResultMap(t, r), nil
	}, inputs)

	require.Len(t, results, 2)
	for i, result := range results {
		assert.Equal(t, strings.ToUpper(inputs[i]), result["TransformedMessage"])
	}
}

// TestBatchKeyedFlushOnSize verifies that keyed batches partition correctly and each key
// flushes independently when its size limit is reached.
func TestBatchKeyedFlushOnSize(t *testing.T) {
	inputs := []KeyedInput{
		{Message: "alpha", Group: "tenant-1"},
		{Message: "bravo", Group: "tenant-1"},
		{Message: "charlie", Group: "tenant-2"},
		{Message: "delta", Group: "tenant-2"},
	}

	results := runConcurrent(t, func(ctx context.Context, inp KeyedInput) (map[string]any, error) {
		r, err := testBatchKeyed.Run(ctx, inp)
		if err != nil {
			return nil, err
		}
		return taskResultMap(t, r), nil
	}, inputs)

	require.Len(t, results, len(inputs))
	for i, result := range results {
		assert.Equal(t, inputs[i].Group, result["batchKey"], "item %d batchKey", i)
		assert.InDelta(t, float64(2), result["batchSize"], 0, "item %d batchSize", i)
		assert.InDelta(t, float64(1), result["uniqueKeys"], 0, "item %d uniqueKeys", i)
		assert.Equal(t, strings.ToUpper(inputs[i].Message), result["uppercase"], "item %d uppercase", i)
	}
}

// TestBatchKeyedFlushOnInterval verifies that keyed batches flush independently per key
// when the interval elapses, even if the per-key size limit is not reached.
func TestBatchKeyedFlushOnInterval(t *testing.T) {
	inputs := []KeyedInput{
		{Message: "echo", Group: "tenant-1"},
		{Message: "foxtrot", Group: "tenant-1"},
		{Message: "golf", Group: "tenant-1"},
		{Message: "hotel", Group: "tenant-2"},
	}

	results := runConcurrent(t, func(ctx context.Context, inp KeyedInput) (map[string]any, error) {
		r, err := testBatchKeyedInterval.Run(ctx, inp)
		if err != nil {
			return nil, err
		}
		return taskResultMap(t, r), nil
	}, inputs)

	require.Len(t, results, len(inputs))

	// First 3 items (tenant-1) should flush together as a batch of size 3.
	for i := 0; i < 3; i++ {
		assert.Equal(t, inputs[i].Group, results[i]["batchKey"], "item %d batchKey", i)
		assert.InDelta(t, float64(3), results[i]["batchSize"], 0, "item %d batchSize", i)
		assert.InDelta(t, float64(1), results[i]["uniqueKeys"], 0, "item %d uniqueKeys", i)
	}

	// Last item (tenant-2) flushes independently after the interval with batchSize=1.
	assert.Equal(t, "tenant-2", results[3]["batchKey"])
	assert.InDelta(t, float64(1), results[3]["batchSize"], 0, "last item batchSize")
	assert.InDelta(t, float64(1), results[3]["uniqueKeys"], 0, "last item uniqueKeys")
	assert.Equal(t, "hotel", results[3]["payload"])
}

// TestBatchLargePayload verifies that batches with large per-item payloads succeed.
func TestBatchLargePayload(t *testing.T) {
	payload := strings.Repeat("x", 4_000_000) // ~4 MB per item
	taskCount := 10

	results := runConcurrent(t, func(ctx context.Context, _ int) (map[string]any, error) {
		r, err := testBatchLarge.Run(ctx, LargeInput{Data: payload})
		if err != nil {
			return nil, err
		}
		return taskResultMap(t, r), nil
	}, make([]int, taskCount))

	require.Len(t, results, taskCount)
	for _, result := range results {
		assert.Equal(t, true, result["received"])
		assert.InDelta(t, float64(taskCount), result["batchSize"], 0)
		assert.InDelta(t, float64(4_000_000), result["dataLength"], 0)
	}
}

// TestBatchSingleItem verifies that a batch task with maxSize=1 processes each item immediately
// as its own batch.
func TestBatchSingleItem(t *testing.T) {
	inputs := []string{"india", "juliet"}

	results := runConcurrent(t, func(ctx context.Context, msg string) (map[string]any, error) {
		r, err := testBatchSingle.Run(ctx, SingleInput{Message: msg})
		if err != nil {
			return nil, err
		}
		return taskResultMap(t, r), nil
	}, inputs)

	require.Len(t, results, 2)
	for i, result := range results {
		assert.InDelta(t, float64(1), result["batchSize"], 0, "item %d batchSize should be 1", i)
		assert.Equal(t, inputs[i], result["original"], "item %d original", i)
	}
}

// TestBatchOrderPreservation verifies that results are returned in submission order
// regardless of batch internal ordering.
func TestBatchOrderPreservation(t *testing.T) {
	count := 20
	indices := make([]int, count)
	for i := range indices {
		indices[i] = i
	}

	rawResults := runConcurrent(t, func(ctx context.Context, idx int) ([]byte, error) {
		r, err := testBatchOrdered.Run(ctx, OrderedInput{Index: idx})
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if err := r.Into(&m); err != nil {
			return nil, err
		}
		return json.Marshal(m)
	}, indices)

	require.Len(t, rawResults, count)
	for i, raw := range rawResults {
		var out map[string]any
		require.NoError(t, json.Unmarshal(raw, &out))
		assert.InDelta(t, float64(i), out["index"], 0, "result %d should have index %d", i, i)
	}
}
