package operation

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testResourceID = uuid.New().String()

func TestInterval_RunInterval_BasicTiming(t *testing.T) {
	interval := &Interval{
		resourceId:      testResourceID,
		maxJitter:       0,
		startInterval:   50 * time.Millisecond,
		currInterval:    50 * time.Millisecond,
		maxInterval:     1 * time.Second,
		noActivityCount: 0,
		incBackoffCount: 3,
		repo:            v1.NewNoOpIntervalSettingsRepository(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	ch := interval.RunInterval(ctx)

	start := time.Now()
	triggerCount := 0

	for {
		select {
		case <-ctx.Done():
			elapsed := time.Since(start)
			assert.GreaterOrEqual(t, triggerCount, 3, "Should have triggered at least 3 times")
			assert.LessOrEqual(t, elapsed, 220*time.Millisecond, "Should complete within timeout plus buffer")
			return
		case <-ch:
			triggerCount++
		}
	}
}

func TestInterval_RunInterval_WithJitter(t *testing.T) {
	interval := &Interval{
		resourceId:      testResourceID,
		maxJitter:       20 * time.Millisecond,
		startInterval:   50 * time.Millisecond,
		currInterval:    50 * time.Millisecond,
		maxInterval:     1 * time.Second,
		noActivityCount: 0,
		incBackoffCount: 3,
		repo:            v1.NewNoOpIntervalSettingsRepository(),
		// firstTrigger left false so subsequent-trigger timing (interval+jitter) is measured
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	ch := interval.RunInterval(ctx)

	var timings []time.Duration
	lastTrigger := time.Now()

	for {
		select {
		case <-ctx.Done():
			assert.GreaterOrEqual(t, len(timings), 2, "Should have at least 2 timing measurements")
			for _, timing := range timings {
				assert.GreaterOrEqual(t, timing, 50*time.Millisecond, "Timing should be at least the base interval")
				assert.LessOrEqual(t, timing, 85*time.Millisecond, "Timing should include jitter but not exceed base + max jitter + buffer")
			}
			return
		case <-ch:
			now := time.Now()
			if len(timings) > 0 || !lastTrigger.IsZero() {
				timings = append(timings, now.Sub(lastTrigger))
			}
			lastTrigger = now
		}
	}
}

func TestInterval_RunInterval_ContextCancellation(t *testing.T) {
	interval := &Interval{
		resourceId:      testResourceID,
		maxJitter:       0,
		startInterval:   100 * time.Millisecond,
		currInterval:    100 * time.Millisecond,
		maxInterval:     1 * time.Second,
		noActivityCount: 0,
		incBackoffCount: 3,
		repo:            v1.NewNoOpIntervalSettingsRepository(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := interval.RunInterval(ctx)

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-ch:
		t.Fatal("Should not receive trigger after context cancellation")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestInterval_SetIntervalGauge_ResetOnRowsModified(t *testing.T) {
	interval := &Interval{
		resourceId:      testResourceID,
		maxJitter:       0,
		startInterval:   50 * time.Millisecond,
		currInterval:    200 * time.Millisecond,
		maxInterval:     1 * time.Second,
		noActivityCount: 5,
		incBackoffCount: 3,
		repo:            v1.NewNoOpIntervalSettingsRepository(),
	}

	interval.SetIntervalGauge(1)

	assert.Equal(t, 50*time.Millisecond, interval.currInterval, "Should reset to start interval")
	assert.Equal(t, 0, interval.noActivityCount, "Should reset no rows count")
}

func TestInterval_SetIntervalGauge_BackoffMechanism(t *testing.T) {
	interval := &Interval{
		resourceId:      testResourceID,
		maxJitter:       0,
		startInterval:   50 * time.Millisecond,
		currInterval:    50 * time.Millisecond,
		maxInterval:     1 * time.Second,
		noActivityCount: 0,
		incBackoffCount: 3,
		repo:            v1.NewNoOpIntervalSettingsRepository(),
	}

	interval.SetIntervalGauge(0)
	assert.Equal(t, 1, interval.noActivityCount)
	assert.Equal(t, 50*time.Millisecond, interval.currInterval)

	interval.SetIntervalGauge(0)
	assert.Equal(t, 2, interval.noActivityCount)
	assert.Equal(t, 50*time.Millisecond, interval.currInterval)

	interval.SetIntervalGauge(0)
	assert.Equal(t, 0, interval.noActivityCount, "Should reset count after backoff")
	assert.Equal(t, 100*time.Millisecond, interval.currInterval, "Should double the interval")

	interval.SetIntervalGauge(0)
	interval.SetIntervalGauge(0)
	interval.SetIntervalGauge(0)
	assert.Equal(t, 200*time.Millisecond, interval.currInterval, "Should double again after 3 more zero-row updates")
}

func TestInterval_SetIntervalGauge_ConcurrentAccess(t *testing.T) {
	interval := &Interval{
		resourceId:      testResourceID,
		maxJitter:       0,
		startInterval:   50 * time.Millisecond,
		currInterval:    50 * time.Millisecond,
		maxInterval:     1 * time.Second,
		noActivityCount: 0,
		incBackoffCount: 3,
		repo:            v1.NewNoOpIntervalSettingsRepository(),
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	numUpdatesPerGoroutine := 50

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numUpdatesPerGoroutine; j++ {
				rowsModified := j % 4
				interval.SetIntervalGauge(rowsModified)
			}
		}(i)
	}

	wg.Wait()

	assert.GreaterOrEqual(t, interval.currInterval, 50*time.Millisecond, "Interval should be at least the start interval")
	assert.GreaterOrEqual(t, interval.noActivityCount, 0, "No rows count should be non-negative")
	assert.LessOrEqual(t, interval.noActivityCount, interval.incBackoffCount-1, "No rows count should not exceed backoff count")
}

func TestInterval_GetNextTrigger_ReturnsChannel(t *testing.T) {
	interval := &Interval{
		resourceId:      testResourceID,
		maxJitter:       10 * time.Millisecond,
		startInterval:   50 * time.Millisecond,
		currInterval:    50 * time.Millisecond,
		maxInterval:     1 * time.Second,
		noActivityCount: 0,
		incBackoffCount: 3,
		repo:            v1.NewNoOpIntervalSettingsRepository(),
	}

	triggerCh := interval.getNextTrigger()
	assert.NotNil(t, triggerCh, "Should return a non-nil channel")

	select {
	case <-triggerCh:
	case <-time.After(70 * time.Millisecond):
		t.Fatal("Trigger should have fired within expected time")
	}
}

func TestInterval_GetNextTrigger_ConcurrentAccess(t *testing.T) {
	interval := &Interval{
		resourceId:      testResourceID,
		maxJitter:       5 * time.Millisecond,
		startInterval:   20 * time.Millisecond,
		currInterval:    20 * time.Millisecond,
		maxInterval:     1 * time.Second,
		noActivityCount: 0,
		incBackoffCount: 3,
		repo:            v1.NewNoOpIntervalSettingsRepository(),
	}

	var wg sync.WaitGroup
	numGoroutines := 5

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				triggerCh := interval.getNextTrigger()
				assert.NotNil(t, triggerCh, "Should always return a non-nil channel")

				select {
				case <-triggerCh:
				case <-time.After(50 * time.Millisecond):
				}
			}
		}()
	}

	wg.Wait()
}

func TestInterval_RunInterval_Integration(t *testing.T) {
	interval := &Interval{
		resourceId:      testResourceID,
		maxJitter:       10 * time.Millisecond,
		startInterval:   50 * time.Millisecond,
		currInterval:    50 * time.Millisecond,
		maxInterval:     1 * time.Second,
		noActivityCount: 0,
		incBackoffCount: 2,
		repo:            v1.NewNoOpIntervalSettingsRepository(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	ch := interval.RunInterval(ctx)

	triggerCount := 0
	for {
		select {
		case <-ctx.Done():
			assert.GreaterOrEqual(t, triggerCount, 3, "Should have triggered multiple times")
			return
		case <-ch:
			triggerCount++

			if triggerCount <= 2 {
				interval.SetIntervalGauge(0)
			} else {
				interval.SetIntervalGauge(1)
			}
		}
	}
}

func TestInterval_GetNextTrigger_FirstTriggerUsesFullWindowPhase(t *testing.T) {
	const (
		base   = 100 * time.Millisecond
		jitter = 50 * time.Millisecond
		window = base + jitter
		n      = 40
	)

	var delays []time.Duration
	for i := 0; i < n; i++ {
		interval := &Interval{
			resourceId:    testResourceID,
			maxJitter:     jitter,
			startInterval: base,
			currInterval:  base,
			maxInterval:   time.Second,
			firstTrigger:  true,
			repo:          v1.NewNoOpIntervalSettingsRepository(),
		}

		start := time.Now()
		<-interval.getNextTrigger()
		delays = append(delays, time.Since(start))

		// Subsequent trigger should wait ~base (+jitter), not a full-window phase near zero.
		start = time.Now()
		<-interval.getNextTrigger()
		second := time.Since(start)
		assert.GreaterOrEqual(t, second, base-5*time.Millisecond, "subsequent trigger should wait at least the base interval")
		assert.LessOrEqual(t, second, window+20*time.Millisecond, "subsequent trigger should not exceed base+jitter")
	}

	var min, max time.Duration = delays[0], delays[0]
	var sum time.Duration
	for _, d := range delays {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
		sum += d
	}

	assert.Less(t, min, 40*time.Millisecond, "first-trigger phase should sometimes be near the start of the window")
	assert.Greater(t, max, 80*time.Millisecond, "first-trigger phase should sometimes land later in the window")
	assert.Less(t, max, window+30*time.Millisecond, "first-trigger phase should stay within the window")
	avg := sum / time.Duration(n)
	assert.Greater(t, avg, 30*time.Millisecond, "average first-trigger delay should be spread across the window")
	assert.Less(t, avg, window, "average first-trigger delay should be below the window upper bound")
}

func TestInterval_NewInterval_DoesNotBlockOnRead(t *testing.T) {
	repo := &countingIntervalRepo{inner: v1.NewNoOpIntervalSettingsRepository()}
	l := zerolog.Nop()

	start := time.Now()
	interval := NewInterval(
		&l,
		repo,
		"timeout-step-runs",
		testResourceID,
		0,
		50*time.Millisecond,
		time.Second,
		3,
		nil,
	)
	elapsed := time.Since(start)

	require.NotNil(t, interval)
	assert.True(t, interval.needsIntervalLoad)
	assert.True(t, interval.firstTrigger)
	assert.Equal(t, int64(0), repo.reads.Load(), "NewInterval must not call ReadInterval")
	assert.Less(t, elapsed, 20*time.Millisecond, "NewInterval should return without waiting on DB")
}

func TestInterval_RunInterval_LazyLoadsPersistedInterval(t *testing.T) {
	persisted := 200 * time.Millisecond
	repo := &countingIntervalRepo{
		inner:      v1.NewNoOpIntervalSettingsRepository(),
		readResult: persisted,
	}
	l := zerolog.Nop()

	interval := NewInterval(
		&l,
		repo,
		"timeout-step-runs",
		testResourceID,
		0,
		50*time.Millisecond,
		time.Second,
		3,
		nil,
	)
	assert.Equal(t, int64(0), repo.reads.Load())
	assert.Equal(t, 50*time.Millisecond, interval.currInterval)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	ch := interval.RunInterval(ctx)

	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("expected at least one trigger after lazy load")
	}

	assert.GreaterOrEqual(t, repo.reads.Load(), int64(1), "RunInterval should lazy-load the persisted interval")
	assert.False(t, interval.needsIntervalLoad)
	assert.Equal(t, persisted, interval.currInterval, "lazy load should apply the persisted interval")
}

func TestInterval_RunInterval_GaugePhaseIsRandomized(t *testing.T) {
	prev := gaugeInterval
	gaugeInterval = 40 * time.Millisecond
	t.Cleanup(func() { gaugeInterval = prev })

	const n = 24
	var (
		mu         sync.Mutex
		firstCalls []time.Duration
		wg         sync.WaitGroup
	)

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	start := time.Now()
	wg.Add(n)

	for i := 0; i < n; i++ {
		interval := &Interval{
			resourceId:      uuid.New().String(),
			maxJitter:       0,
			startInterval:   time.Hour, // keep method triggers out of the way
			currInterval:    time.Hour,
			maxInterval:     time.Hour,
			incBackoffCount: 3,
			repo:            v1.NewNoOpIntervalSettingsRepository(),
			firstTrigger:    true,
			gauge: func(context.Context, string) (int, error) {
				mu.Lock()
				firstCalls = append(firstCalls, time.Since(start))
				mu.Unlock()
				wg.Done()
				// Only record the first call per interval: replace gauge after first fire
				// by returning quickly; subsequent ticks may race Done, so use Once per interval.
				return 0, nil
			},
		}

		// Wrap gauge with Once so wg.Done is only called once per interval.
		var once sync.Once
		g := interval.gauge
		interval.gauge = func(ctx context.Context, resourceId string) (int, error) {
			once.Do(func() {
				_, _ = g(ctx, resourceId)
			})
			return 0, nil
		}

		_ = interval.RunInterval(ctx)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("timed out waiting for gauge first calls; got %d/%d", len(firstCalls), n)
	}

	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, len(firstCalls), n)

	var min, max time.Duration = firstCalls[0], firstCalls[0]
	for _, d := range firstCalls[:n] {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	// Without phase spread, all first gauge ticks would cluster near gaugeInterval.
	// With phase in [0, gaugeInterval) then a full tick, first calls land in
	// roughly [gaugeInterval, 2*gaugeInterval). Spread across that window.
	assert.Greater(t, max-min, 15*time.Millisecond, "gauge first-call times should be phase-spread, not synchronized")
}

type countingIntervalRepo struct {
	inner      v1.IntervalSettingsRepository
	readResult time.Duration
	reads      atomic.Int64
}

func (r *countingIntervalRepo) ReadAllIntervals(ctx context.Context, operationId string) (map[string]time.Duration, error) {
	return r.inner.ReadAllIntervals(ctx, operationId)
}

func (r *countingIntervalRepo) ReadInterval(ctx context.Context, operationId string, tenantId uuid.UUID) (time.Duration, error) {
	r.reads.Add(1)
	if r.readResult > 0 {
		return r.readResult, nil
	}
	return r.inner.ReadInterval(ctx, operationId, tenantId)
}

func (r *countingIntervalRepo) SetInterval(ctx context.Context, operationId string, tenantId uuid.UUID, d time.Duration) (time.Duration, error) {
	return r.inner.SetInterval(ctx, operationId, tenantId, d)
}
