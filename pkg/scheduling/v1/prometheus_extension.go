package v1

import (
	"sync"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type PrometheusExtension struct {
	mu                     sync.RWMutex
	tenants                map[uuid.UUID]*sqlcv1.Tenant
	tenantIdToWorkerLabels map[uuid.UUID]map[WorkerPromLabels]struct{}
}

func NewPrometheusExtension() *PrometheusExtension {
	return &PrometheusExtension{
		tenants:                make(map[uuid.UUID]*sqlcv1.Tenant),
		tenantIdToWorkerLabels: make(map[uuid.UUID]map[WorkerPromLabels]struct{}),
	}
}

func (p *PrometheusExtension) SetTenants(tenants []*sqlcv1.Tenant) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, tenant := range tenants {
		p.tenants[tenant.ID] = tenant
	}
}

type WorkerPromLabels struct {
	ID   uuid.UUID
	Name string
}

func (p *PrometheusExtension) ReportSnapshot(tenantId uuid.UUID, input *SnapshotInput) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tenantIdStr := tenantId.String()

	workerPromLabelsToSlotData := make(map[WorkerPromLabels]*SlotUtilization)

	for workerId, utilization := range input.WorkerSlotUtilization {
		worker, ok := input.Workers[workerId]
		if !ok {
			continue
		}

		promLabels := WorkerPromLabels{
			ID:   worker.WorkerId,
			Name: worker.Name,
		}

		data, ok := workerPromLabelsToSlotData[promLabels]
		if ok {
			data.UtilizedSlots += utilization.UtilizedSlots
			data.NonUtilizedSlots += utilization.NonUtilizedSlots
		} else {
			workerPromLabelsToSlotData[promLabels] = &SlotUtilization{
				UtilizedSlots:    utilization.UtilizedSlots,
				NonUtilizedSlots: utilization.NonUtilizedSlots,
			}
		}
	}

	if known, ok := p.tenantIdToWorkerLabels[tenantId]; ok {
		for labels := range known {
			if _, stillActive := workerPromLabelsToSlotData[labels]; !stillActive {
				prometheus.TenantWorkerSlots.DeleteLabelValues(tenantIdStr, labels.ID.String(), labels.Name)
				prometheus.TenantUsedWorkerSlots.DeleteLabelValues(tenantIdStr, labels.ID.String(), labels.Name)
				prometheus.TenantAvailableWorkerSlots.DeleteLabelValues(tenantIdStr, labels.ID.String(), labels.Name)
			}
		}
	}

	currentWorkers := make(map[WorkerPromLabels]struct{}, len(workerPromLabelsToSlotData))
	for promLabels, utilization := range workerPromLabelsToSlotData {
		usedSlots := float64(utilization.UtilizedSlots)
		availableSlots := float64(utilization.NonUtilizedSlots)

		prometheus.TenantWorkerSlots.WithLabelValues(tenantIdStr, promLabels.ID.String(), promLabels.Name).Set(usedSlots + availableSlots)
		prometheus.TenantUsedWorkerSlots.WithLabelValues(tenantIdStr, promLabels.ID.String(), promLabels.Name).Set(usedSlots)
		prometheus.TenantAvailableWorkerSlots.WithLabelValues(tenantIdStr, promLabels.ID.String(), promLabels.Name).Set(availableSlots)

		currentWorkers[promLabels] = struct{}{}
	}

	p.tenantIdToWorkerLabels[tenantId] = currentWorkers
}

func (p *PrometheusExtension) PostAssign(tenantId uuid.UUID, input *PostAssignInput) {}

func (p *PrometheusExtension) Cleanup() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.tenants = make(map[uuid.UUID]*sqlcv1.Tenant)
	p.tenantIdToWorkerLabels = make(map[uuid.UUID]map[WorkerPromLabels]struct{})
	return nil
}
