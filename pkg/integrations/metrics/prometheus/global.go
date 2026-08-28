package prometheus

import (
	"math"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type GlobalHatchetMetric string

const (
	QueueInvocationsTotal       GlobalHatchetMetric = "hatchet_queue_invocations_total"
	CreatedTasksTotal           GlobalHatchetMetric = "hatchet_created_tasks_total"
	RetriedTasksTotal           GlobalHatchetMetric = "hatchet_retried_tasks_total"
	SucceededTasksTotal         GlobalHatchetMetric = "hatchet_succeeded_tasks_total"
	FailedTasksTotal            GlobalHatchetMetric = "hatchet_failed_tasks_total"
	SkippedTasksTotal           GlobalHatchetMetric = "hatchet_skipped_tasks_total"
	CancelledTasksTotal         GlobalHatchetMetric = "hatchet_cancelled_tasks_total"
	AssignedTasksTotal          GlobalHatchetMetric = "hatchet_assigned_tasks"
	SchedulingTimedOutTotal     GlobalHatchetMetric = "hatchet_scheduling_timed_out"
	RateLimitedTotal            GlobalHatchetMetric = "hatchet_rate_limited"
	QueuedToAssignedTotal       GlobalHatchetMetric = "hatchet_queued_to_assigned"
	QueuedToAssignedTimeSeconds GlobalHatchetMetric = "hatchet_queued_to_assigned_time_seconds"
	ReassignedTasksTotal        GlobalHatchetMetric = "hatchet_reassigned_tasks"

	SchedulingLatencyP50Seconds          GlobalHatchetMetric = "hatchet_scheduling_latency_p50_seconds"
	SchedulingLatencyP99Seconds          GlobalHatchetMetric = "hatchet_scheduling_latency_p99_seconds"
	SchedulingLatencySamplesDroppedTotal GlobalHatchetMetric = "hatchet_scheduling_latency_samples_dropped_total"
)

var (
	QueueInvocations = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(QueueInvocationsTotal),
		Help: "The total number of invocations of the queuer function",
	})

	CreatedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(CreatedTasksTotal),
		Help: "The total number of tasks created",
	})

	RetriedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(RetriedTasksTotal),
		Help: "The total number of tasks retried",
	})

	SucceededTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(SucceededTasksTotal),
		Help: "The total number of tasks that succeeded",
	})

	FailedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(FailedTasksTotal),
		Help: "The total number of tasks that failed (in a final state, not including retries)",
	})

	SkippedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(SkippedTasksTotal),
		Help: "The total number of tasks that were skipped",
	})

	CancelledTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(CancelledTasksTotal),
		Help: "The total number of tasks cancelled",
	})

	AssignedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(AssignedTasksTotal),
		Help: "The total number of tasks assigned to a worker",
	})

	SchedulingTimedOut = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(SchedulingTimedOutTotal),
		Help: "The total number of tasks that timed out while waiting to be scheduled",
	})

	RateLimited = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(RateLimitedTotal),
		Help: "The total number of tasks that were rate limited",
	})

	QueuedToAssigned = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(QueuedToAssignedTotal),
		Help: "The total number of unique tasks that were queued and later got assigned to a worker",
	})

	QueuedToAssignedTimeBuckets = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    string(QueuedToAssignedTimeSeconds),
		Help:    "Buckets of time in seconds spent in the queue before being assigned to a worker",
		Buckets: []float64{0.01, 0.02, 0.05, 0.1, 0.5, 1, 2, 5, 15},
	})

	ReassignedTasks = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(ReassignedTasksTotal),
		Help: "The total number of tasks that were reassigned to a worker",
	})

	// The scheduling latency gauges start (and idle) at NaN rather than 0: a
	// registered gauge always renders, and 0 would read as "instant
	// scheduling" on pods that schedule nothing. client_golang renders NaN
	// literally and Prometheus stores it as a non-value, producing gaps.
	SchedulingLatencyP50 = newNaNGauge(prometheus.GaugeOpts{
		Name: string(SchedulingLatencyP50Seconds),
		Help: "True p50 of queued-to-assigned time over a 30s sliding window of this process's samples; NaN when the window is empty. Unlike hatchet_queued_to_assigned_time_seconds, not clamped by histogram buckets.",
	})

	SchedulingLatencyP99 = newNaNGauge(prometheus.GaugeOpts{
		Name: string(SchedulingLatencyP99Seconds),
		Help: "True p99 of queued-to-assigned time over a 30s sliding window of this process's samples; NaN when the window is empty. Unlike hatchet_queued_to_assigned_time_seconds, not clamped by histogram buckets.",
	})

	SchedulingLatencySamplesDropped = promauto.NewCounter(prometheus.CounterOpts{
		Name: string(SchedulingLatencySamplesDroppedTotal),
		Help: "Scheduling latency samples dropped because one second exceeded the quantile window's per-second capacity; nonzero means the latency quantile gauges under-sample peak seconds.",
	})
)

func newNaNGauge(opts prometheus.GaugeOpts) prometheus.Gauge {
	g := promauto.NewGauge(opts)
	g.Set(math.NaN())
	return g
}
