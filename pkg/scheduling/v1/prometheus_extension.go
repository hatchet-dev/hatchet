package v1

import (
	"sync"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type PrometheusExtension struct {
	mu      sync.RWMutex
	tenants map[string]*dbsqlc.Tenant
}

func NewPrometheusExtension() *PrometheusExtension {
	return &PrometheusExtension{
		tenants: make(map[string]*dbsqlc.Tenant),
	}
}

func (p *PrometheusExtension) SetTenants(tenants []*dbsqlc.Tenant) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, tenant := range tenants {
		p.tenants[tenant.ID.String()] = tenant
	}
}

func (p *PrometheusExtension) ReportSnapshot(tenantId string, input *SnapshotInput) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	workerNameToSlotData := make(map[string]*SlotUtilization)

	for workerId, utilization := range input.WorkerSlotUtilization {
		worker, ok := input.Workers[workerId]
		if !ok {
			continue
		}

		data, ok := workerNameToSlotData[worker.Name]
		if ok {
			data.UtilizedSlots += utilization.UtilizedSlots
			data.NonUtilizedSlots += utilization.NonUtilizedSlots
			workerNameToSlotData[worker.Name] = data
		} else {
			workerNameToSlotData[worker.Name] = &SlotUtilization{
				UtilizedSlots:    utilization.UtilizedSlots,
				NonUtilizedSlots: utilization.NonUtilizedSlots,
			}
		}
	}

	for workerName, utilization := range workerNameToSlotData {
		totalSlots := float64(utilization.UtilizedSlots + utilization.NonUtilizedSlots)
		usedSlots := float64(utilization.UtilizedSlots)
		availableSlots := float64(utilization.NonUtilizedSlots)

		prometheus.TenantWorkerSlots.WithLabelValues(workerName).Set(totalSlots)
		prometheus.TenantUsedWorkerSlots.WithLabelValues(workerName).Set(usedSlots)
		prometheus.TenantAvailableWorkerSlots.WithLabelValues(workerName).Set(availableSlots)
	}
}

func (p *PrometheusExtension) PostAssign(tenantId string, input *PostAssignInput) {}

func (p *PrometheusExtension) Cleanup() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.tenants = make(map[string]*dbsqlc.Tenant)
	return nil
}
