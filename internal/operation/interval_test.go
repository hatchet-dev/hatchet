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

// The timing tests in this file run against the real clock. Under -race on a
// shared CI runner, timers have been observed firing 20-30ms late, so the
// assertions follow two rules:
//
//   - Lower bounds are strict. time.After never fires early, and every
//     measurement window is opened before the timer it measures is armed.
//   - Upper bounds are relative to the interval under test and leave at least
//     half an interval of headroom (about 3x the observed lateness), so a real
//     regression such as a doubled interval or a phase drawn from a doubled
//     window still fails.
//
// For repeating triggers the upper bound is checked cumulatively at the last
// trigger (see assertTriggerCadence), so one stalled timer is absorbed by the
// headroom of the other cycles while a systematically slow interval still fails.

// triggerTimes runs the interval until ctx is done and returns the elapsed
// time of every trigger, measured from before RunInterval arms its first
// timer. Each timer is armed only after the previous trigger was delivered, so
// the k-th (1-based) trigger cannot arrive before k times the minimum delay,
// no matter how the goroutines are scheduled.
func triggerTimes(ctx context.Context, interval *Interval) []time.Duration {
	start := time.Now()
	ch := interval.RunInterval(ctx)

	var times []time.Duration
	for {
		select {
		case <-ctx.Done():
			return times
		case <-ch:
			times = append(times, time.Since(start))
		}
	}
}

// assertTriggerCadence checks that no trigger arrived faster than minGap per
// cycle (strict) and that the last trigger has not drifted past maxGap per
// cycle on average.
func assertTriggerCadence(t *testing.T, times []time.Duration, minGap, maxGap time.Duration) {
	t.Helper()

	require.GreaterOrEqual(t, len(times), 3, "interval should keep triggering for the whole run")

	for k, elapsed := range times {
		assert.GreaterOrEqual(t, elapsed, time.Duration(k+1)*minGap, "trigger %d arrived faster than %d cycles of %v allow", k+1, k+1, minGap)
	}

	n := len(times)
	assert.LessOrEqual(t, times[n-1], time.Duration(n)*maxGap, "trigger %d arrived later than %d cycles of %v allow", n, n, maxGap)
}

func TestInterval_RunInterval_BasicTiming(t *testing.T) {
	const (
		base = 100 * time.Millisecond
		run  = 1 * time.Second
	)

	interval := &Interval{
		resourceId:      testResourceID,
		maxJitter:       0,
		startInterval:   base,
		currInterval:    base,
		maxInterval:     1 * time.Second,
		noActivityCount: 0,
		incBackoffCount: 3,
		repo:            v1.NewNoOpIntervalSettingsRepository(),
		// firstTrigger left false so every cycle, including the first, waits a full base interval.
	}

	ctx, cancel := context.WithTimeout(context.Background(), run)
	defer cancel()

	times := triggerTimes(ctx, interval)

	// A doubled interval would put the last trigger at >= 2*base per cycle.
	assertTriggerCadence(t, times, base, base+base/2)
}

func TestInterval_RunInterval_WithJitter(t *testing.T) {
	const (
		base   = 100 * time.Millisecond
		jitter = 50 * time.Millisecond
		window = base + jitter
		run    = 1 * time.Second
	)

	interval := &Interval{
		resourceId:      testResourceID,
		maxJitter:       jitter,
		startInterval:   base,
		currInterval:    base,
		maxInterval:     1 * time.Second,
		noActivityCount: 0,
		incBackoffCount: 3,
		repo:            v1.NewNoOpIntervalSettingsRepository(),
		// firstTrigger left false so subsequent-trigger timing (interval+jitter) is measured
	}

	ctx, cancel := context.WithTimeout(context.Background(), run)
	defer cancel()

	times := triggerTimes(ctx, interval)

	// Jitter only ever adds to the base interval, and never more than maxJitter.
	assertTriggerCadence(t, times, base, window+window/2)
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
	// A single timer gets no averaging, so use a larger interval than the
	// cadence tests: the half-window headroom is then 150ms of lateness.
	const (
		base   = 200 * time.Millisecond
		jitter = 100 * time.Millisecond
		window = base + jitter
	)

	interval := &Interval{
		resourceId:      testResourceID,
		maxJitter:       jitter,
		startInterval:   base,
		currInterval:    base,
		maxInterval:     1 * time.Second,
		noActivityCount: 0,
		incBackoffCount: 3,
		repo:            v1.NewNoOpIntervalSettingsRepository(),
	}

	start := time.Now()
	triggerCh := interval.getNextTrigger()
	require.NotNil(t, triggerCh, "Should return a non-nil channel")

	select {
	case <-triggerCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Trigger never fired")
	}
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, base, "trigger should wait at least the base interval")
	assert.LessOrEqual(t, elapsed, window+window/2, "trigger should not wait much longer than base+jitter")
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
	// Every sample is a single timer with no averaging, and the bounds below
	// look at the extremes of 40 of them, so use a larger interval than the
	// cadence tests: the half-window headroom is then 150ms of lateness.
	const (
		base   = 200 * time.Millisecond
		jitter = 100 * time.Millisecond
		window = base + jitter
		// late is the timer lateness tolerated on upper bounds. Half a window
		// keeps a phase drawn from a doubled window (max near 2*window) failing.
		late = window / 2
		n    = 40
	)

	// The samples are independent, so take them in parallel: sequentially this
	// test took about 8s, which is a long time to hold a CI test slot. Each
	// sample's clock starts before its timer is armed, so scheduling delay
	// between the two counts against the upper bounds; bounding how many
	// samples run at once keeps the delay this test inflicts on itself small.
	const parallelism = 8
	var (
		wg      sync.WaitGroup
		slots   = make(chan struct{}, parallelism)
		firsts  = make([]time.Duration, n)
		seconds = make([]time.Duration, n)
	)

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()

			slots <- struct{}{}
			defer func() { <-slots }()

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
			firsts[i] = time.Since(start)

			start = time.Now()
			<-interval.getNextTrigger()
			seconds[i] = time.Since(start)
		}(i)
	}
	wg.Wait()

	// Subsequent triggers should wait base (+jitter), not a full-window phase near zero.
	for i, second := range seconds {
		assert.GreaterOrEqual(t, second, base, "subsequent trigger %d should wait at least the base interval", i)
		assert.LessOrEqual(t, second, window+late, "subsequent trigger %d should not exceed base+jitter", i)
	}

	var min, max time.Duration = firsts[0], firsts[0]
	var sum time.Duration
	for _, d := range firsts {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
		sum += d
	}

	// With 40 samples drawn uniformly from [0, window), the chance that none
	// lands in a given half of the window is 2^-40, so these bounds hold even
	// when every timer runs tens of milliseconds late.
	assert.Less(t, min, window/2, "first-trigger phase should sometimes land in the first half of the window")
	assert.Greater(t, max, window/2, "first-trigger phase should sometimes land in the second half of the window")
	assert.Less(t, max, window+late, "first-trigger phase should stay within the window")
	avg := sum / time.Duration(n)
	assert.Greater(t, avg, window/5, "average first-trigger delay should be spread across the window")
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
	// The zero read count above is the real check; this bound only guards
	// against the constructor blocking outright, so it is deliberately loose.
	assert.Less(t, elapsed, time.Second, "NewInterval should return without waiting on DB")
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

	// The first trigger lands within [0, persisted); the timeout only bounds
	// the failure path, so it is deliberately loose.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	// Large enough that a scheduler stall of 100ms or more can neither
	// collapse the phase spread of the samples nor push the latest first
	// call past the half-interval headroom of the upper bound below.
	gaugeInterval = 400 * time.Millisecond
	t.Cleanup(func() { gaugeInterval = prev })

	const n = 24
	var (
		mu         sync.Mutex
		firstCalls []time.Duration
		wg         sync.WaitGroup
	)

	// First calls land within 2*gaugeInterval; the timeout only bounds the
	// failure path, so it is deliberately loose.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	case <-ctx.Done():
		mu.Lock()
		got := len(firstCalls)
		mu.Unlock()
		t.Fatalf("timed out waiting for gauge first calls; got %d/%d", got, n)
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

	// With phase in [0, gaugeInterval) followed by a full tick, first calls
	// land in [gaugeInterval, 2*gaugeInterval). Without phase spread they would
	// all cluster just after gaugeInterval.
	assert.GreaterOrEqual(t, min, gaugeInterval, "gauge must not be called before a full tick has elapsed")
	assert.Less(t, max, 2*gaugeInterval+gaugeInterval/2, "gauge first calls should land within phase plus one tick")
	assert.Greater(t, max-min, gaugeInterval/4, "gauge first-call times should be phase-spread, not synchronized")
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
