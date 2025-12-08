package scheduler

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	repov1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func newNoopLogger() *zerolog.Logger {
	l := zerolog.New(io.Discard).With().Timestamp().Logger()
	return &l
}

func TestBatchBufferManager_FlushOnBatchSize(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var mu sync.Mutex
	flushed := make([]*batchFlushRequest, 0)

	manager := newBatchBufferManager(newNoopLogger(), func(_ context.Context, req *batchFlushRequest) error {
		mu.Lock()
		defer mu.Unlock()
		flushed = append(flushed, req)
		return nil
	}, nil)

	cfg := batchConfig{
		batchSize:     2,
		flushInterval: 0,
	}

	item1 := &repov1.AssignedItem{
		QueueItem: &sqlcv1.V1QueueItem{
			TaskID:     1,
			RetryCount: 0,
		},
	}

	item2 := &repov1.AssignedItem{
		QueueItem: &sqlcv1.V1QueueItem{
			TaskID:     2,
			RetryCount: 0,
		},
	}

	addResult, err := manager.Add(ctx, "tenant", "step-1", "workflow:task", "dispatcher", "worker-a", "batch", cfg, item1)
	require.NoError(t, err)
	assert.False(t, addResult.Flushed)
	assert.Equal(t, 1, addResult.Pending)
	assert.Nil(t, addResult.FlushReason)
	assert.NotEmpty(t, addResult.PendingBatchID)
	assert.Empty(t, addResult.FlushedBatchID)
	firstBatchID := addResult.PendingBatchID

	addResult, err = manager.Add(ctx, "tenant", "step-1", "workflow:task", "dispatcher", "worker-a", "batch", cfg, item2)
	require.NoError(t, err)
	assert.True(t, addResult.Flushed)
	assert.Equal(t, 0, addResult.Pending)
	if assert.NotNil(t, addResult.FlushReason) {
		assert.Equal(t, flushReasonBatchSizeReached, *addResult.FlushReason)
	}
	assert.NotEmpty(t, addResult.FlushedBatchID)
	assert.NotEmpty(t, addResult.PendingBatchID)
	assert.NotEqual(t, firstBatchID, addResult.PendingBatchID, "batch id should rotate after flush")

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(flushed) == 1
	}, 100*time.Millisecond, 10*time.Millisecond, "expected flush to occur")

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, flushed, 1)
	require.Equal(t, "worker-a", flushed[0].WorkerID)
	require.Equal(t, "dispatcher", flushed[0].DispatcherID)
	require.Len(t, flushed[0].Items, 2)
	assert.Equal(t, flushReasonBatchSizeReached, flushed[0].FlushReason)
	assert.Equal(t, cfg.batchSize, flushed[0].ConfiguredBatchSize)
	assert.False(t, flushed[0].TriggeredAt.IsZero())
	assert.Equal(t, firstBatchID, flushed[0].BatchID)
	assert.Equal(t, addResult.FlushedBatchID, flushed[0].BatchID)
}

func TestBatchBufferManager_FlushOnInterval(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	flushCh := make(chan *batchFlushRequest, 1)

	manager := newBatchBufferManager(newNoopLogger(), func(_ context.Context, req *batchFlushRequest) error {
		flushCh <- req
		return nil
	}, nil)

	cfg := batchConfig{
		batchSize:     10,
		flushInterval: 50 * time.Millisecond,
	}

	item := &repov1.AssignedItem{
		QueueItem: &sqlcv1.V1QueueItem{
			TaskID:     10,
			RetryCount: 0,
		},
	}

	addResult, err := manager.Add(ctx, "tenant", "step-1", "workflow:task", "dispatcher", "worker-a", "batch", cfg, item)
	require.NoError(t, err)
	assert.False(t, addResult.Flushed)
	assert.Equal(t, 1, addResult.Pending)
	assert.NotNil(t, addResult.NextFlushAt)
	assert.NotEmpty(t, addResult.PendingBatchID)
	initialBatchID := addResult.PendingBatchID

	select {
	case req := <-flushCh:
		require.Equal(t, "worker-a", req.WorkerID)
		require.Len(t, req.Items, 1)
		assert.Equal(t, flushReasonIntervalElapsed, req.FlushReason)
		assert.Equal(t, cfg.flushInterval, req.ConfiguredFlushInterval)
		assert.Equal(t, initialBatchID, req.BatchID)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected flush from timer")
	}
}

func TestBatchBufferManager_WorkerChangeDoesNotFlushImmediately(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var mu sync.Mutex
	flushed := make([]*batchFlushRequest, 0)

	manager := newBatchBufferManager(newNoopLogger(), func(_ context.Context, req *batchFlushRequest) error {
		mu.Lock()
		defer mu.Unlock()
		flushed = append(flushed, req)
		return nil
	}, nil)

	cfg := batchConfig{
		batchSize:     5,
		flushInterval: 0,
	}

	newItem := func(id int64) *repov1.AssignedItem {
		return &repov1.AssignedItem{
			QueueItem: &sqlcv1.V1QueueItem{
				TaskID:     id,
				RetryCount: 0,
			},
		}
	}

	firstResult, err := manager.Add(ctx, "tenant", "step-1", "workflow:task", "dispatcher", "worker-a", "batch", cfg, newItem(1))
	require.NoError(t, err)
	require.NotEmpty(t, firstResult.PendingBatchID)

	secondResult, err := manager.Add(ctx, "tenant", "step-1", "workflow:task", "dispatcher", "worker-a", "batch", cfg, newItem(2))
	require.NoError(t, err)
	assert.Equal(t, firstResult.PendingBatchID, secondResult.PendingBatchID)

	addResult, err := manager.Add(ctx, "tenant", "step-1", "workflow:task", "dispatcher", "worker-b", "batch", cfg, newItem(3))
	require.NoError(t, err)
	assert.False(t, addResult.Flushed, "worker change should not flush previous buffer")
	assert.Equal(t, 1, addResult.Pending, "new worker should start a fresh buffer")
	assert.NotEmpty(t, addResult.PendingBatchID)
	assert.NotEqual(t, firstResult.PendingBatchID, addResult.PendingBatchID)
	assert.Empty(t, addResult.FlushedBatchID)

	for _, id := range []int64{4, 5, 6} {
		_, err := manager.Add(ctx, "tenant", "step-1", "workflow:task", "dispatcher", "worker-a", "batch", cfg, newItem(id))
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(flushed) == 1
	}, 100*time.Millisecond, 10*time.Millisecond, "expected flush after worker-a batch filled")

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, flushed, 1)
	assert.Equal(t, "worker-a", flushed[0].WorkerID)
	require.Len(t, flushed[0].Items, cfg.batchSize)
	assert.Equal(t, flushReasonBatchSizeReached, flushed[0].FlushReason)
	assert.Equal(t, cfg.batchSize, flushed[0].ConfiguredBatchSize)
	assert.Equal(t, firstResult.PendingBatchID, flushed[0].BatchID)
}

func TestBatchBufferManager_UniqueBatchIDsAcrossSequentialFlushes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var mu sync.Mutex
	var flushed []*batchFlushRequest

	manager := newBatchBufferManager(newNoopLogger(), func(_ context.Context, req *batchFlushRequest) error {
		mu.Lock()
		defer mu.Unlock()
		flushed = append(flushed, req)
		return nil
	}, nil)

	cfg := batchConfig{
		batchSize:     2,
		flushInterval: 0,
	}

	newItem := func(id int64) *repov1.AssignedItem {
		return &repov1.AssignedItem{
			QueueItem: &sqlcv1.V1QueueItem{
				TaskID:     id,
				RetryCount: 0,
			},
		}
	}

	for _, id := range []int64{1, 2, 3, 4} {
		_, err := manager.Add(ctx, "tenant", "step-1", "workflow:task", "dispatcher", "worker-a", "group", cfg, newItem(id))
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(flushed) == 2
	}, 100*time.Millisecond, 10*time.Millisecond, "expected two flushes to occur")

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, flushed, 2)
	require.NotEmpty(t, flushed[0].BatchID)
	require.NotEmpty(t, flushed[1].BatchID)
	assert.NotEqual(t, flushed[0].BatchID, flushed[1].BatchID, "each flush must have a unique batch id")
}
