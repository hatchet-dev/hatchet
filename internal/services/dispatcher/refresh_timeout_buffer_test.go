package dispatcher

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshTimeoutBufferSumsPending(t *testing.T) {
	b := newRefreshTimeoutBuffer()
	key := refreshTimeoutKey{
		tenantId:       uuid.New(),
		taskExternalId: uuid.New(),
	}

	b.add(key, time.Second)
	b.add(key, 2*time.Second)

	sum, ok := b.take(key)
	require.True(t, ok)
	assert.Equal(t, 3*time.Second, sum)

	_, ok = b.take(key)
	assert.False(t, ok)
}

func TestRefreshTimeoutBufferDebounceAndTrailingEstimate(t *testing.T) {
	b := newRefreshTimeoutBuffer()
	key := refreshTimeoutKey{
		tenantId:       uuid.New(),
		taskExternalId: uuid.New(),
	}

	assert.True(t, b.shouldFlush(key))

	b.add(key, time.Second)
	sum, ok := b.take(key)
	require.True(t, ok)
	assert.Equal(t, time.Second, sum)

	flushedAt := time.Now().Add(5 * time.Second)
	b.markFlushed(key, flushedAt)
	assert.False(t, b.shouldFlush(key))

	b.add(key, 2*time.Second)
	estimate, ok := b.lastTimeout(key)
	require.True(t, ok)
	assert.Equal(t, flushedAt.Add(2*time.Second), estimate)
}

// Simulates the singleflight race: take() sees empty, then a waiter add()'s
// before the leader returns. hasPending must observe the new increment.
func TestRefreshTimeoutBufferHasPendingAfterEmptyTake(t *testing.T) {
	b := newRefreshTimeoutBuffer()
	key := refreshTimeoutKey{
		tenantId:       uuid.New(),
		taskExternalId: uuid.New(),
	}

	_, ok := b.take(key)
	assert.False(t, ok)
	assert.False(t, b.hasPending(key))

	b.add(key, time.Second)
	assert.True(t, b.hasPending(key))

	sum, ok := b.take(key)
	require.True(t, ok)
	assert.Equal(t, time.Second, sum)
}
