package operation

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/stretchr/testify/assert"
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
				assert.LessOrEqual(t, timing, 75*time.Millisecond, "Timing should include jitter but not exceed base + max jitter + buffer")
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
