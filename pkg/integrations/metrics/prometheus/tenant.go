package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type TenantHatchetMetric string

const (
	TenantWorkflowDurationMilliseconds TenantHatchetMetric = "tenant_workflow_duration_milliseconds"
	TenantAssignedTasksTotal           TenantHatchetMetric = "tenant_assigned_tasks"
	TenantSchedulingTimedOutTotal      TenantHatchetMetric = "tenant_scheduling_timed_out"
	TenantRateLimitedTotal             TenantHatchetMetric = "tenant_rate_limited"
	TenantQueuedToAssignedTotal        TenantHatchetMetric = "tenant_queued_to_assigned"
	TenantQueuedToAssignedTimeSeconds  TenantHatchetMetric = "tenant_queued_to_assigned_time_seconds"
)

var (
	TenantWorkflowDurationBuckets = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: string(TenantWorkflowDurationMilliseconds),
		Help: "Duration of workflow execution in milliseconds (DAGs and single tasks)",
		Buckets: []float64{
			// Sub-millisecond and very fast
			0.1, // 0.1ms
			0.5, // 0.5ms
			1,   // 1ms
			2,   // 2ms
			5,   // 5ms
			10,  // 10ms
			25,  // 25ms
			50,  // 50ms
			100, // 100ms
			250, // 250ms
			500, // 500ms

			// Seconds
			1000,  // 1s
			2500,  // 2.5s
			5000,  // 5s
			10000, // 10s
			30000, // 30s
			60000, // 1min

			// Minutes to hours
			300000,   // 5min
			1800000,  // 30min
			3600000,  // 1hr
			10800000, // 3hr
			21600000, // 6hr
			43200000, // 12hr
			86400000, // 24hr
		},
	}, []string{"tenant_id", "workflow_name", "status"})

	TenantAssignedTasks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: string(TenantAssignedTasksTotal),
		Help: "The total number of tasks assigned to a worker",
	}, []string{"tenant_id"})

	TenantSchedulingTimedOut = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: string(TenantSchedulingTimedOutTotal),
		Help: "The total number of tasks that timed out while waiting to be scheduled",
	}, []string{"tenant_id"})

	TenantRateLimited = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: string(TenantRateLimitedTotal),
		Help: "The total number of tasks that were rate limited",
	}, []string{"tenant_id"})

	TenantQueuedToAssigned = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: string(TenantQueuedToAssignedTotal),
		Help: "The total number of unique tasks that were queued and later got assigned to a worker",
	}, []string{"tenant_id"})

	TenantQueuedToAssignedTimeBuckets = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    string(TenantQueuedToAssignedTimeSeconds),
		Help:    "Buckets of time in seconds spent in the queue before being assigned to a worker",
		Buckets: []float64{0.01, 0.02, 0.05, 0.1, 0.5, 1, 2, 5, 15},
	}, []string{"tenant_id"})
)
