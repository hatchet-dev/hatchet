package operation

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharedGauge_ReusesResultPerResourceWithinTick(t *testing.T) {
	// Uses the production tick so no entry can expire while the test runs.
	calls := map[string]int{}
	gauge := NewSharedGauge(func(_ context.Context, resourceId string) (int, error) {
		calls[resourceId]++
		return len(resourceId), nil
	})

	first, second := uuid.New().String(), "other-resource"

	// eight callers, as with the eight task controller pools
	for range 8 {
		res, err := gauge(context.Background(), first)
		require.NoError(t, err)
		assert.Equal(t, len(first), res)
	}

	res, err := gauge(context.Background(), second)
	require.NoError(t, err)
	assert.Equal(t, len(second), res, "a different resource must not get another resource's result")

	assert.Equal(t, map[string]int{first: 1, second: 1}, calls, "each resource should be queried once per tick")
}

func TestSharedGauge_QueriesAgainAfterTick(t *testing.T) {
	prev := gaugeInterval
	gaugeInterval = 20 * time.Millisecond
	t.Cleanup(func() { gaugeInterval = prev })

	var calls atomic.Int64
	gauge := NewSharedGauge(func(context.Context, string) (int, error) {
		return int(calls.Add(1)), nil
	})

	res, err := gauge(context.Background(), testResourceID)
	require.NoError(t, err)
	assert.Equal(t, 1, res)

	// Sleeping can only overshoot, so the entry is certain to have expired.
	time.Sleep(3 * gaugeInterval)

	res, err = gauge(context.Background(), testResourceID)
	require.NoError(t, err)
	assert.Equal(t, 2, res, "a result older than one tick must not be reused")
}

func TestSharedGauge_DoesNotKeepErrors(t *testing.T) {
	var calls int
	gauge := NewSharedGauge(func(context.Context, string) (int, error) {
		calls++
		if calls == 1 {
			return 0, errors.New("database unavailable")
		}
		return 1, nil
	})

	_, err := gauge(context.Background(), testResourceID)
	require.Error(t, err)

	res, err := gauge(context.Background(), testResourceID)
	require.NoError(t, err)
	assert.Equal(t, 1, res, "the call after an error should reach the underlying gauge")
	assert.Equal(t, 2, calls)
}
