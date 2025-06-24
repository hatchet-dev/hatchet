package prometheus

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type TenantHatchetMetric string

const (
	TenantWorkflowCompleted TenantHatchetMetric = "hatchet_workflow_completed"
)

type TenantMetric struct {
	WorkflowCompleted *prometheus.CounterVec
}

var tenantMetricsMap sync.Map

func WithTenant(tenantId string) *TenantMetric {
	tenantMetric, ok := tenantMetricsMap.Load(tenantId)
	if !ok {
		workflowCompleted := promauto.NewCounterVec(prometheus.CounterOpts{
			Name: string(TenantWorkflowCompleted),
			Help: "Finished workflow runs",
		}, []string{"tenant_id", "workflow_name", "status", "duration", "worker_id"})

		prometheus.MustRegister(workflowCompleted)

		tenantMetric = &TenantMetric{
			WorkflowCompleted: workflowCompleted,
		}

		tenantMetricsMap.Store(tenantId, tenantMetric)
	}

	return tenantMetric.(*TenantMetric)
}
