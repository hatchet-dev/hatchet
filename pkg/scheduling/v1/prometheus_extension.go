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
		currentWorkers[promLabels] = struct{}{}

		usedSlots := float64(utilization.UtilizedSlots)
		availableSlots := float64(utilization.NonUtilizedSlots)

		// Skip setting gauge values for workers with no slots yet to avoid reporting
		// transient 0-slot state between worker registration and slot replenishment.
		// Previous non-zero values are preserved until the worker is replenished or removed.
		if usedSlots+availableSlots == 0 {
			continue
		}

		prometheus.TenantWorkerSlots.WithLabelValues(tenantIdStr, promLabels.ID.String(), promLabels.Name).Set(usedSlots + availableSlots)
		prometheus.TenantUsedWorkerSlots.WithLabelValues(tenantIdStr, promLabels.ID.String(), promLabels.Name).Set(usedSlots)
		prometheus.TenantAvailableWorkerSlots.WithLabelValues(tenantIdStr, promLabels.ID.String(), promLabels.Name).Set(availableSlots)
	}

	p.tenantIdToWorkerLabels[tenantId] = currentWorkers
}

func (p *PrometheusExtension) PostAssign(tenantId uuid.UUID, input *PostAssignInput) {}

func (p *PrometheusExtension) deleteGaugesForTenant(tenantId uuid.UUID) {
	tenantIdStr := tenantId.String()

	if known, ok := p.tenantIdToWorkerLabels[tenantId]; ok {
		for labels := range known {
			prometheus.TenantWorkerSlots.DeleteLabelValues(tenantIdStr, labels.ID.String(), labels.Name)
			prometheus.TenantUsedWorkerSlots.DeleteLabelValues(tenantIdStr, labels.ID.String(), labels.Name)
			prometheus.TenantAvailableWorkerSlots.DeleteLabelValues(tenantIdStr, labels.ID.String(), labels.Name)
		}

		delete(p.tenantIdToWorkerLabels, tenantId)
	}

	delete(p.tenants, tenantId)
}

func (p *PrometheusExtension) CleanupTenant(tenantId uuid.UUID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.deleteGaugesForTenant(tenantId)
	return nil
}

func (p *PrometheusExtension) Cleanup() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for tenantId := range p.tenantIdToWorkerLabels {
		p.deleteGaugesForTenant(tenantId)
	}

	p.tenants = make(map[uuid.UUID]*sqlcv1.Tenant)
	p.tenantIdToWorkerLabels = make(map[uuid.UUID]map[WorkerPromLabels]struct{})
	return nil
}
