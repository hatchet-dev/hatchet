package prometheus

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type TenantHatchetMetric string

const (
	TenantWorkflowDuration            TenantHatchetMetric = "hatchet_workflow_duration_milliseconds"
	TenantAssignedTasks               TenantHatchetMetric = "hatchet_assigned_tasks"
	TenantSchedulingTimedOut          TenantHatchetMetric = "hatchet_scheduling_timed_out"
	TenantRateLimited                 TenantHatchetMetric = "hatchet_rate_limited"
	TenantQueuedToAssigned            TenantHatchetMetric = "hatchet_queued_to_assigned"
	TenantQueuedToAssignedTimeBuckets TenantHatchetMetric = "hatchet_queued_to_assigned_time_seconds"
)

type TenantMetric struct {
	WorkflowDuration            *prometheus.HistogramVec
	AssignedTasks               *prometheus.CounterVec
	SchedulingTimedOut          *prometheus.CounterVec
	RateLimited                 *prometheus.CounterVec
	QueuedToAssigned            *prometheus.CounterVec
	QueuedToAssignedTimeBuckets *prometheus.HistogramVec
}

var tenantMetricsMap sync.Map

func WithTenant(tenantId string) *TenantMetric {
	tenantMetric, ok := tenantMetricsMap.Load(tenantId)
	if !ok {
		workflowDuration := promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: string(TenantWorkflowDuration),
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

		assignedTasks := promauto.NewCounterVec(prometheus.CounterOpts{
			Name: string(TenantAssignedTasks),
			Help: "The total number of tasks assigned to a worker",
		}, []string{"tenant_id"})

		schedulingTimedOut := promauto.NewCounterVec(prometheus.CounterOpts{
			Name: string(TenantSchedulingTimedOut),
			Help: "The total number of tasks that timed out while waiting to be scheduled",
		}, []string{"tenant_id"})

		rateLimited := promauto.NewCounterVec(prometheus.CounterOpts{
			Name: string(TenantRateLimited),
			Help: "The total number of tasks that were rate limited",
		}, []string{"tenant_id"})

		queuedToAssigned := promauto.NewCounterVec(prometheus.CounterOpts{
			Name: string(TenantQueuedToAssigned),
			Help: "The total number of unique tasks that were queued and later got assigned to a worker",
		}, []string{"tenant_id"})

		queuedToAssignedTimeBuckets := promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    string(TenantQueuedToAssignedTimeBuckets),
			Help:    "Buckets of time in seconds spent in the queue before being assigned to a worker",
			Buckets: []float64{0.01, 0.02, 0.05, 0.1, 0.5, 1, 2, 5, 15},
		}, []string{"tenant_id"})

		tenantMetric = &TenantMetric{
			WorkflowDuration:            workflowDuration,
			AssignedTasks:               assignedTasks,
			SchedulingTimedOut:          schedulingTimedOut,
			RateLimited:                 rateLimited,
			QueuedToAssigned:            queuedToAssigned,
			QueuedToAssignedTimeBuckets: queuedToAssignedTimeBuckets,
		}

		tenantMetricsMap.Store(tenantId, tenantMetric)
	}

	return tenantMetric.(*TenantMetric)
}
