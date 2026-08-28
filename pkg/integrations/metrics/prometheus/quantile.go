package prometheus

import (
	"math"
	"sort"
	"sync"
	"time"
)

const (
	// schedulingLatencyWindowSeconds is the sliding window over which the
	// scheduling latency quantile gauges are computed.
	schedulingLatencyWindowSeconds = 30

	// schedulingLatencyPerSecondCap bounds window memory: at most
	// windowSeconds * perSecondCap float64 samples (~8 MiB at 30 * 34k).
	// When a single second exceeds the cap, further samples for that second
	// are dropped and counted in
	// hatchet_scheduling_latency_samples_dropped_total (drop-newest is the
	// simplest predictable degradation; sample order within one second
	// carries little information, so the quantiles stay representative).
	schedulingLatencyPerSecondCap = 34_000
)

// quantileWindow tracks a sliding window of observations as a ring of
// per-second sample slices. Observe is a cheap append under a mutex; quantile
// computation snapshots the in-window samples and sorts outside the lock.
type quantileWindow struct {
	mu sync.Mutex

	// buckets[i] holds the samples for unix second secs[i]; a bucket is
	// reused (reset to length zero, keeping capacity) when its ring slot is
	// claimed by a new second.
	buckets [][]float64
	secs    []int64

	windowSecs   int
	perSecondCap int

	now    func() time.Time
	onDrop func()
}

func newQuantileWindow(windowSecs, perSecondCap int, now func() time.Time, onDrop func()) *quantileWindow {
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
		onDrop:       onDrop,
	}
}

func (w *quantileWindow) Observe(v float64) {
	sec := w.now().Unix()
	idx := int(sec % int64(w.windowSecs))

	w.mu.Lock()

	if w.secs[idx] != sec {
		w.buckets[idx] = w.buckets[idx][:0]
		w.secs[idx] = sec
	}

	if len(w.buckets[idx]) >= w.perSecondCap {
		w.mu.Unlock()
		if w.onDrop != nil {
			w.onDrop()
		}
		return
	}

	w.buckets[idx] = append(w.buckets[idx], v)

	w.mu.Unlock()
}

// quantiles returns the p50 and p99 of the samples observed within the
// window, or (NaN, NaN) when the window is empty.
func (w *quantileWindow) quantiles() (p50 float64, p99 float64) {
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
		return math.NaN(), math.NaN()
	}

	snapshot := make([]float64, 0, total)
	for i := range w.buckets {
		if w.secs[i] >= oldest {
			snapshot = append(snapshot, w.buckets[i]...)
		}
	}

	w.mu.Unlock()

	sort.Float64s(snapshot)

	return quantileSorted(snapshot, 0.5), quantileSorted(snapshot, 0.99)
}

// quantileSorted returns the nearest-rank quantile of a sorted, non-empty
// slice.
func quantileSorted(sorted []float64, q float64) float64 {
	rank := int(math.Ceil(q*float64(len(sorted)))) - 1
	if rank < 0 {
		rank = 0
	}
	if rank >= len(sorted) {
		rank = len(sorted) - 1
	}
	return sorted[rank]
}

var (
	schedulingLatencyWindow = newQuantileWindow(
		schedulingLatencyWindowSeconds,
		schedulingLatencyPerSecondCap,
		time.Now,
		SchedulingLatencySamplesDropped.Inc,
	)

	schedulingLatencyLoopOnce sync.Once
)

// ObserveSchedulingLatency records a queued-to-assigned duration for the true
// quantile gauges (hatchet_scheduling_latency_p50_seconds / _p99_seconds).
// It exists alongside hatchet_queued_to_assigned_time_seconds because that
// histogram's top bucket is 15s, so its estimated p99 clamps during incidents;
// these gauges sort the actual samples.
//
// The first call starts a process-lifetime goroutine that recomputes the
// quantiles once per second, so processes that never schedule (API pods,
// other engine roles) pay nothing and their gauges stay NaN. NaN (the
// registered gauges' initial value, see global.go) is what client_golang
// renders when the window is empty: Prometheus stores it as a gap-producing
// non-value, unlike a misleading 0.
func ObserveSchedulingLatency(seconds float64) {
	schedulingLatencyLoopOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(time.Second)
			for range ticker.C {
				p50, p99 := schedulingLatencyWindow.quantiles()
				SchedulingLatencyP50.Set(p50)
				SchedulingLatencyP99.Set(p99)
			}
		}()
	})

	schedulingLatencyWindow.Observe(seconds)
}
