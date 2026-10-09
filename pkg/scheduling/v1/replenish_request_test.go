//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// newRelease returns a channel that blocks fake database reads and a function
// that closes it. The channel is also closed when the test ends, so a failed
// assertion does not leave a cycle blocked on it.
func newRelease(t *testing.T) (<-chan struct{}, func()) {
	t.Helper()

	release := make(chan struct{})
	closeRelease := sync.OnceFunc(func() { close(release) })
	t.Cleanup(closeRelease)

	return release, closeRelease
}

// startReplenishLoop runs loopReplenish until the test ends, with ticks pushed
// out of the test's reach so every cycle it sees is request-driven.
func startReplenishLoop(t *testing.T, s *Scheduler) {
	t.Helper()

	s.replenishTickerMin = time.Hour
	s.replenishTickerMax = 2 * time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		s.loopReplenish(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		<-done
	})
}

func TestScheduler_RequestReplenishDoesNotWaitForCycle(t *testing.T) {
	var cycles atomic.Int64
	release, closeRelease := newRelease(t)

	s := newTestScheduler(t, uuid.New(), &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			cycles.Add(1)
			select {
			case <-release:
			case <-ctx.Done():
			}
			return nil, nil
		},
	})
	startReplenishLoop(t, s)

	s.notifyReplenish()
	require.Eventually(t, func() bool { return cycles.Load() == 1 }, time.Second, time.Millisecond)

	// the cycle is blocked on its database read; requests still return at once
	start := time.Now()
	for range 1000 {
		s.notifyReplenish()
	}
	assert.Less(t, time.Since(start), 100*time.Millisecond)

	closeRelease()

	// the 1000 requests made during the cycle merge into one follow-up cycle
	require.Eventually(t, func() bool { return cycles.Load() == 2 }, time.Second, time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	assert.EqualValues(t, 2, cycles.Load())
}

func TestScheduler_RequestReplenishMergesBurstAndServesLastRequest(t *testing.T) {
	const (
		cycleTime = 20 * time.Millisecond
		burst     = 300 * time.Millisecond
	)

	var mu sync.Mutex
	var cycleStarts []time.Time

	s := newTestScheduler(t, uuid.New(), &mockAssignmentRepo{
		listActionsForWorkersFn: func(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
			mu.Lock()
			cycleStarts = append(cycleStarts, time.Now())
			mu.Unlock()
			time.Sleep(cycleTime)
			return nil, nil
		},
	})
	startReplenishLoop(t, s)

	requests := 0
	var lastRequest time.Time

	for start := time.Now(); time.Since(start) < burst; {
		lastRequest = time.Now()
		s.notifyReplenish()
		requests++
		time.Sleep(time.Millisecond)
	}

	numCycles := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(cycleStarts)
	}

	// once the burst is over the loop goes quiet
	require.Eventually(t, func() bool {
		n := numCycles()
		time.Sleep(3 * cycleTime)
		return numCycles() == n
	}, time.Second, time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	t.Logf("%d requests ran %d replenish cycles", requests, len(cycleStarts))

	// cycles run back to back while requests keep arriving, never one per request
	assert.GreaterOrEqual(t, len(cycleStarts), 2)
	assert.LessOrEqual(t, len(cycleStarts), 2*int(burst/cycleTime))
	assert.Less(t, len(cycleStarts), requests)

	// a request is never dropped: some cycle starts after the last one
	assert.False(t, cycleStarts[len(cycleStarts)-1].Before(lastRequest), "no replenish cycle started after the last request")
}
