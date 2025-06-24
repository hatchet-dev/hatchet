package prometheus

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type TenantHatchetMetric string

const (
	TenantWorkflowCompleted TenantHatchetMetric = "hatchet_workflow_completed"
	TenantWorkflowDuration  TenantHatchetMetric = "hatchet_workflow_duration_milliseconds"
)

type TenantMetric struct {
	WorkflowCompleted *prometheus.CounterVec
	WorkflowDuration  *prometheus.HistogramVec
}

var tenantMetricsMap sync.Map

func WithTenant(tenantId string) *TenantMetric {
	tenantMetric, ok := tenantMetricsMap.Load(tenantId)
	if !ok {
		workflowCompleted := promauto.NewCounterVec(prometheus.CounterOpts{
			Name: string(TenantWorkflowCompleted),
			Help: "Finished workflow runs",
		}, []string{"tenant_id", "workflow_name", "status"})

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

		tenantMetric = &TenantMetric{
			WorkflowCompleted: workflowCompleted,
			WorkflowDuration:  workflowDuration,
		}

		tenantMetricsMap.Store(tenantId, tenantMetric)
	}

	return tenantMetric.(*TenantMetric)
}
