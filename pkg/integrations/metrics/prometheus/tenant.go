package prometheus

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type TenantHatchetMetric string

const (
	TenantWorkflowCompleted TenantHatchetMetric = "hatchet_workflow_completed"
)

var tenantMap sync.Map

type TenantMetrics struct {
	WorkflowCompleted *prometheus.CounterVec

	Registry *prometheus.Registry
}

func WithTenant(tenantId string) *TenantMetrics {
	tenantMetrics, ok := tenantMap.Load(tenantId)
	if !ok {
		registry := prometheus.NewRegistry()

		workflowCompleted := prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: string(TenantWorkflowCompleted),
			Help: "The total number of workflows completed",
		}, []string{"tenant_id", "workflow_id", "worker_id", "status"})

		registry.MustRegister(workflowCompleted)

		tenantMetrics = &TenantMetrics{
			WorkflowCompleted: workflowCompleted,
			Registry:          registry,
		}

		tenantMap.Store(tenantId, tenantMetrics)
	}
	return tenantMetrics.(*TenantMetrics)
}
