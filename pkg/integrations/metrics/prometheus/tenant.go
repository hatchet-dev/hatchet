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
	Registry          *prometheus.Registry
	WorkflowCompleted *prometheus.CounterVec
}

var tenantMetricsMap sync.Map

func WithTenant(tenantId string) *TenantMetric {
	tenantMetric, ok := tenantMetricsMap.Load(tenantId)
	if !ok {
		registry := prometheus.NewRegistry()

		workflowCompleted := promauto.NewCounterVec(prometheus.CounterOpts{
			Name: string(TenantWorkflowCompleted),
			Help: "Finished workflow runs",
		}, []string{"tenant_id", "workflow_name", "status", "duration", "worker_id"})

		registry.MustRegister(workflowCompleted)

		tenantMetric = &TenantMetric{
			Registry:          registry,
			WorkflowCompleted: workflowCompleted,
		}

		tenantMetricsMap.Store(tenantId, tenantMetric)
	}

	return tenantMetric.(*TenantMetric)
}
