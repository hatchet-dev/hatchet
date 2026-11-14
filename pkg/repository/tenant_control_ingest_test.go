//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIngestBucketSteadyRate(t *testing.T) {
	start := time.Unix(0, 0)
	b := newIngestBucket(10)

	// a full bucket admits one second of single requests at once
	for i := range 10 {
		require.Zero(t, b.take(1, start), "request %d", i)
	}

	require.Equal(t, 100*time.Millisecond, b.take(1, start), "the next token is a tenth of a second away")

	// then one request per tenth of a second: 100 over the next 10 seconds
	admitted := 0

	for now := start; now.Before(start.Add(10 * time.Second)); now = now.Add(10 * time.Millisecond) {
		if b.take(1, now.Add(10*time.Millisecond)) == 0 {
			admitted++
		}
	}

	require.Equal(t, 100, admitted)
}

func TestIngestBucketAdmitsAWholeRequestOrNothing(t *testing.T) {
	start := time.Unix(0, 0)
	b := newIngestBucket(10)

	require.Zero(t, b.take(7, start))
	require.Equal(t, 200*time.Millisecond, b.take(5, start), "5 items need 2 more tokens")
	require.Zero(t, b.take(3, start), "a refused request takes no tokens")
	require.Equal(t, 500*time.Millisecond, b.take(5, start))
	require.Zero(t, b.take(5, start.Add(500*time.Millisecond)))
}

func TestIngestBucketLargeBatchIsAdmittedWhenFull(t *testing.T) {
	start := time.Unix(0, 0)
	b := newIngestBucket(10)

	require.Zero(t, b.take(1, start))
	require.Equal(t, 100*time.Millisecond, b.take(30, start), "a batch over one second of tokens waits for a full bucket")

	now := start.Add(100 * time.Millisecond)
	require.Zero(t, b.take(30, now), "then it is admitted whole")

	// it left 20 tokens of debt: the next single item waits 2.1 seconds, a full batch 3 seconds
	require.Equal(t, 2100*time.Millisecond, b.take(1, now))
	require.Equal(t, 3*time.Second, b.take(30, now))
	require.Zero(t, b.take(30, now.Add(3*time.Second)))
}
