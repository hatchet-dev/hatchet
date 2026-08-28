package prometheus

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// fakeClock lets tests control which per-second bucket receives observations.
type fakeClock struct {
	t time.Time
}

func (c *fakeClock) now() time.Time {
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.t = c.t.Add(d)
}

func newTestWindow(windowSecs, perSecondCap int) (*quantileWindow, *fakeClock) {
	clock := &fakeClock{t: time.Unix(1_000_000, 0)}
	return newQuantileWindow(windowSecs, perSecondCap, clock.now), clock
}

func TestQuantileWindowKnownDistribution(t *testing.T) {
	w, clock := newTestWindow(30, 1000)

	assert.True(t, math.IsNaN(w.quantile(0.5)), "empty window must not report a number")
	assert.True(t, math.IsNaN(w.quantile(0.99)), "empty window must not report a number")

	// 1..100 spread over several seconds: nearest-rank p50 is the 50th value,
	// p99 the 99th.
	for i := 1; i <= 100; i++ {
		w.Observe(float64(i))
		if i%10 == 0 {
			clock.advance(time.Second)
		}
	}

	assert.Equal(t, 50.0, w.quantile(0.5))
	assert.Equal(t, 99.0, w.quantile(0.99))
}

func TestQuantileWindowExpiry(t *testing.T) {
	w, clock := newTestWindow(30, 1000)

	w.Observe(100.0) // old sample

	clock.advance(10 * time.Second)
	w.Observe(1.0)
	w.Observe(2.0)
	w.Observe(3.0)

	// Both generations in the window: the old 100 is the max, so p99 = 100.
	assert.Equal(t, 100.0, w.quantile(0.99))

	// 25s after the old sample it is still in the 30s window...
	clock.advance(15 * time.Second)
	assert.Equal(t, 100.0, w.quantile(0.99))

	// ...but 35s after, only the newer samples remain.
	clock.advance(10 * time.Second)
	assert.Equal(t, 2.0, w.quantile(0.5))
	assert.Equal(t, 3.0, w.quantile(0.99))

	// Once everything ages out the window is empty again.
	clock.advance(30 * time.Second)
	assert.True(t, math.IsNaN(w.quantile(0.5)))
	assert.True(t, math.IsNaN(w.quantile(0.99)))
}

func TestQuantileWindowPerSecondCap(t *testing.T) {
	w, clock := newTestWindow(30, 3)

	// Only the first 3 samples of the second are kept.
	for i := 1; i <= 5; i++ {
		w.Observe(float64(i))
	}

	assert.Equal(t, 2.0, w.quantile(0.5))
	assert.Equal(t, 3.0, w.quantile(0.99))

	// The cap is per second: the next second accepts samples again.
	clock.advance(time.Second)
	w.Observe(10.0)

	assert.Equal(t, 10.0, w.quantile(0.99))
}

func TestQuantileWindowRingSlotReuse(t *testing.T) {
	windowSecs := 5
	w, clock := newTestWindow(windowSecs, 1000)

	w.Observe(100.0)

	// windowSecs later the same ring slot is claimed by a new second; the old
	// bucket must be discarded, not mixed in.
	clock.advance(time.Duration(windowSecs) * time.Second)
	w.Observe(1.0)

	assert.Equal(t, 1.0, w.quantile(0.5))
	assert.Equal(t, 1.0, w.quantile(0.99))
}
