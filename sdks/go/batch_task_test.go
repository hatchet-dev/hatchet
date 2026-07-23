//go:build !e2e && !load && !rampup && !integration

package hatchet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleBatchTaskFn is a minimal function matching the (non-broadcast) batch task signature.
func sampleBatchTaskFn(_ Context, _ map[string]any) (map[string]any, error) { return nil, nil }

// sampleBroadcastBatchTaskFn is a minimal function matching the broadcast batch task signature.
func sampleBroadcastBatchTaskFn(_ Context, _ map[string]any) (any, error) { return nil, nil }

func TestNewStandaloneBatchTask_SerializesBatchConfig(t *testing.T) {
	maxInterval := 200 * time.Millisecond
	groupKey := "input.group"
	groupMaxRuns := int32(3)

	c := newTestClient()

	task := c.NewStandaloneBatchTask("batch-task", sampleBatchTaskFn, BatchConfig{
		MaxSize:         5,
		MaxInterval:     &maxInterval,
		GroupKey:        &groupKey,
		GroupMaxRuns:    &groupMaxRuns,
		BroadcastOutput: false,
	})

	req, _, _, _ := task.Dump()

	require.Len(t, req.Tasks, 1)

	taskOpts := req.Tasks[0]

	require.NotNil(t, taskOpts.Batch)
	assert.Equal(t, int32(5), taskOpts.Batch.BatchMaxSize)
	require.NotNil(t, taskOpts.Batch.BatchMaxIntervalMs)
	assert.Equal(t, int32(200), *taskOpts.Batch.BatchMaxIntervalMs)
	require.NotNil(t, taskOpts.Batch.BatchGroupKey)
	assert.Equal(t, "input.group", *taskOpts.Batch.BatchGroupKey)
	require.NotNil(t, taskOpts.Batch.BatchGroupMaxRuns)
	assert.Equal(t, int32(3), *taskOpts.Batch.BatchGroupMaxRuns)
	assert.False(t, taskOpts.Batch.GetBroadcastOutput())

	// retries must always be forced to 0 for batch tasks.
	assert.Equal(t, int32(0), taskOpts.Retries)
}

func TestNewStandaloneBatchTask_BroadcastOutput(t *testing.T) {
	c := newTestClient()

	task := c.NewStandaloneBatchTask("batch-broadcast-task", sampleBroadcastBatchTaskFn, BatchConfig{
		MaxSize:         10,
		BroadcastOutput: true,
	})

	req, _, _, _ := task.Dump()

	require.Len(t, req.Tasks, 1)

	taskOpts := req.Tasks[0]

	require.NotNil(t, taskOpts.Batch)
	assert.Equal(t, int32(10), taskOpts.Batch.BatchMaxSize)
	assert.Nil(t, taskOpts.Batch.BatchMaxIntervalMs)
	assert.Nil(t, taskOpts.Batch.BatchGroupKey)
	assert.True(t, taskOpts.Batch.GetBroadcastOutput())
}

func TestNewStandaloneBatchTask_RetriesOverridesAreIgnored(t *testing.T) {
	c := newTestClient()

	// Even if a caller supplies WithRetries, batch tasks always force retries to 0.
	task := c.NewStandaloneBatchTask("batch-task-with-retries", sampleBatchTaskFn, BatchConfig{
		MaxSize: 5,
	}, WithRetries(3))

	req, _, _, _ := task.Dump()

	require.Len(t, req.Tasks, 1)
	assert.Equal(t, int32(0), req.Tasks[0].Retries)
}

func TestNewBatchTask_PanicsOnInvalidConfig(t *testing.T) {
	tests := []struct {
		name  string
		batch BatchConfig
	}{
		{name: "zero max size", batch: BatchConfig{MaxSize: 0}},
		{name: "negative max size", batch: BatchConfig{MaxSize: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient()
			workflow := c.NewWorkflow("batch-invalid-config")

			assert.Panics(t, func() {
				workflow.NewBatchTask("batch-task", sampleBatchTaskFn, tt.batch)
			})
		})
	}
}

func TestNewBatchTask_PanicsOnNonMapOutputWithoutBroadcast(t *testing.T) {
	c := newTestClient()
	workflow := c.NewWorkflow("batch-invalid-output")

	badFn := func(_ Context, _ map[string]any) (string, error) { return "", nil }

	assert.Panics(t, func() {
		workflow.NewBatchTask("batch-task", badFn, BatchConfig{MaxSize: 5})
	})
}

func TestNewBatchTask_PanicsOnDurable(t *testing.T) {
	c := newTestClient()
	workflow := c.NewWorkflow("batch-durable")

	assert.Panics(t, func() {
		workflow.NewBatchTask("batch-task", sampleBatchTaskFn, BatchConfig{MaxSize: 5}, withDurable())
	})
}
