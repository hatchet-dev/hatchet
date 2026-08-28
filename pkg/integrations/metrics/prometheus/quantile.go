package prometheus

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	schedulingLatencyWindowSeconds = 30

	// Bounds window memory at windowSeconds * cap float64 samples (~24 MiB).
	// Sized well above the highest assignment rate seen in production
	// (~94k/s), so silently discarding a second's excess samples is
	// effectively unreachable.
	schedulingLatencyPerSecondCap = 100_000
)

// quantileWindow tracks a sliding window of observations as a ring of
// per-second sample slices.
type quantileWindow struct {
	mu      sync.Mutex
	buckets [][]float64
	secs    []int64 // secs[i] is the unix second buckets[i] currently holds

	windowSecs   int
	perSecondCap int

	now func() time.Time
}

func newQuantileWindow(windowSecs, perSecondCap int, now func() time.Time) *quantileWindow {
	secs := make([]int64, windowSecs)
	for i := range secs {
		secs[i] = -1
	}

	return &quantileWindow{
		buckets:      make([][]float64, windowSecs),
		secs:         secs,
		windowSecs:   windowSecs,
		perSecondCap: perSecondCap,
		now:          now,
	}
}

func (w *quantileWindow) Observe(v float64) {
	sec := w.now().Unix()
	idx := int(sec % int64(w.windowSecs))

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.secs[idx] != sec {
		// the slot's previous second aged out of the window; discard its
		// samples but keep the bucket's capacity
		w.buckets[idx] = w.buckets[idx][:0]
		w.secs[idx] = sec
	}

	if len(w.buckets[idx]) < w.perSecondCap {
		w.buckets[idx] = append(w.buckets[idx], v)
	}
}

// quantile returns the nearest-rank q-quantile (0 < q <= 1) of the samples in
// the window, or NaN when the window is empty.
func (w *quantileWindow) quantile(q float64) float64 {
	oldest := w.now().Unix() - int64(w.windowSecs) + 1

	w.mu.Lock()

	total := 0
	for i := range w.buckets {
		if w.secs[i] >= oldest {
			total += len(w.buckets[i])
		}
	}

	if total == 0 {
		w.mu.Unlock()
		return math.NaN()
	}

	snapshot := make([]float64, 0, total)
	for i := range w.buckets {
		if w.secs[i] >= oldest {
			snapshot = append(snapshot, w.buckets[i]...)
		}
	}

	w.mu.Unlock()

	sort.Float64s(snapshot)

	return snapshot[int(math.Ceil(q*float64(len(snapshot))))-1]
}

var schedulingLatencyWindow = newQuantileWindow(
	schedulingLatencyWindowSeconds,
	schedulingLatencyPerSecondCap,
	time.Now,
)

// The gauges compute quantiles lazily at scrape time. Processes that never
// schedule report NaN, which client_golang renders literally and Prometheus
// stores as a gap — not a misleading 0.
var (
	_ = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "hatchet_scheduling_latency_p50_seconds",
		Help: "True p50 of queued-to-assigned time over a 30s sliding window of this process's samples; NaN when the window is empty. Unlike hatchet_queued_to_assigned_time_seconds, not clamped by histogram buckets.",
	}, func() float64 { return schedulingLatencyWindow.quantile(0.5) })

	_ = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "hatchet_scheduling_latency_p99_seconds",
		Help: "True p99 of queued-to-assigned time over a 30s sliding window of this process's samples; NaN when the window is empty. Unlike hatchet_queued_to_assigned_time_seconds, not clamped by histogram buckets.",
	}, func() float64 { return schedulingLatencyWindow.quantile(0.99) })
)

// ObserveSchedulingLatency records a queued-to-assigned duration for the true
// quantile gauges. They exist alongside hatchet_queued_to_assigned_time_seconds
// because that histogram's 15s top bucket clamps its estimated p99 during
// incidents.
func ObserveSchedulingLatency(seconds float64) {
	schedulingLatencyWindow.Observe(seconds)
}
