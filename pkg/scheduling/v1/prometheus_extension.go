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

type WorkerPromLabels struct {
	ID   string
	Name string
}

func (p *PrometheusExtension) ReportSnapshot(tenantId string, input *SnapshotInput) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	workerPromLabelsToSlotData := make(map[*WorkerPromLabels]*SlotUtilization)

	for workerId, utilization := range input.WorkerSlotUtilization {
		worker, ok := input.Workers[workerId]
		if !ok {
			continue
		}

		promLabels := &WorkerPromLabels{
			ID:   worker.WorkerId,
			Name: worker.Name,
		}

		data, ok := workerPromLabelsToSlotData[promLabels]
		if ok {
			data.UtilizedSlots += utilization.UtilizedSlots
			data.NonUtilizedSlots += utilization.NonUtilizedSlots
			workerPromLabelsToSlotData[promLabels] = data
		} else {
			workerPromLabelsToSlotData[promLabels] = utilization
		}
	}

	for promLabels, utilization := range workerPromLabelsToSlotData {
		totalSlots := float64(utilization.TotalSlots)
		usedSlots := float64(utilization.UtilizedSlots)
		availableSlots := float64(utilization.NonUtilizedSlots)

		prometheus.TenantWorkerSlots.WithLabelValues(promLabels.ID, promLabels.Name).Set(totalSlots)
		prometheus.TenantUsedWorkerSlots.WithLabelValues(promLabels.ID, promLabels.Name).Set(usedSlots)
		prometheus.TenantAvailableWorkerSlots.WithLabelValues(promLabels.ID, promLabels.Name).Set(availableSlots)
	}
}

func (p *PrometheusExtension) PostAssign(tenantId string, input *PostAssignInput) {}

func (p *PrometheusExtension) Cleanup() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.tenants = make(map[string]*dbsqlc.Tenant)
	return nil
}
