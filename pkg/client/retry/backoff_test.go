package retry

import (
	"context"
	"math/rand/v2"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFullJitterDelayDeterministic(t *testing.T) {
	t.Parallel()

	r := rand.New(rand.NewPCG(1, 1))
	delay := fullJitterDelay(2, restBaseDelay, restBackoffFactor, restMaxDelay, r.Int64N)
	assert.GreaterOrEqual(t, delay, time.Duration(0))
	assert.LessOrEqual(t, delay, restMaxDelay)
}

func TestParseRetryAfterDeltaSeconds(t *testing.T) {
	t.Parallel()

	now := time.Unix(0, 0).UTC()

	delay, ok := parseRetryAfter("2", now, restMaxRetryAfter)
	require.True(t, ok)
	assert.Equal(t, 2*time.Second, delay)
}

func TestParseRetryAfterHTTPDate(t *testing.T) {
	t.Parallel()

	now := time.Unix(0, 0).UTC()
	retryAt := now.Add(2 * time.Second).UTC().Format(http.TimeFormat)

	delay, ok := parseRetryAfter(retryAt, now, restMaxRetryAfter)
	require.True(t, ok)
	assert.Equal(t, 2*time.Second, delay)
}

func TestParseRetryAfterPastHTTPDate(t *testing.T) {
	t.Parallel()

	now := time.Unix(100, 0).UTC()
	past := now.Add(-5 * time.Second).UTC().Format(http.TimeFormat)

	delay, ok := parseRetryAfter(past, now, restMaxRetryAfter)
	require.True(t, ok)
	assert.Equal(t, time.Duration(0), delay)
}

func TestParseRetryAfterInvalidAndOversized(t *testing.T) {
	t.Parallel()

	now := time.Unix(0, 0).UTC()

	_, ok := parseRetryAfter("not-a-date", now, restMaxRetryAfter)
	assert.False(t, ok)

	_, ok = parseRetryAfter("10", now, restMaxRetryAfter)
	assert.False(t, ok)

	_, ok = parseRetryAfter("-1", now, restMaxRetryAfter)
	assert.False(t, ok)
}

func TestSleepContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := sleepContext(ctx, time.Second)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSleepContextZeroDelay(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	err := sleepContext(ctx, 0)
	assert.NoError(t, err)
}
