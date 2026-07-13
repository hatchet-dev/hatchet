package v1

import (
	"context"
	"strconv"
	"sync"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type PrometheusExtension struct {
	mu                     sync.RWMutex
	tenants                map[uuid.UUID]*sqlcv1.Tenant
	tenantIdToWorkerLabels map[uuid.UUID]map[WorkerPromLabels]struct{}
	tenantIdToLabelPairs   map[uuid.UUID]map[LabelPairPromLabels]struct{}

	promGate *prometheus.Gate
}

func NewPrometheusExtension(promGate *prometheus.Gate) *PrometheusExtension {
	return &PrometheusExtension{
		tenants:                make(map[uuid.UUID]*sqlcv1.Tenant),
		tenantIdToWorkerLabels: make(map[uuid.UUID]map[WorkerPromLabels]struct{}),
		tenantIdToLabelPairs:   make(map[uuid.UUID]map[LabelPairPromLabels]struct{}),
		promGate:               promGate,
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

type WorkerLabelPair struct {
	Key   string
	Value string
}

// LabelPairPromLabels identifies a label-pair slot series: one worker label pair
// combined with one slot type.
type LabelPairPromLabels struct {
	WorkerLabelPair
	SlotType string
}

func workerLabelPairs(labels []*sqlcv1.ListManyWorkerLabelsRow) []WorkerLabelPair {
	pairs := make([]WorkerLabelPair, 0, len(labels))
	seen := make(map[WorkerLabelPair]struct{}, len(labels))

	for _, label := range labels {
		var value string

		switch {
		case label.StrValue.Valid:
			value = label.StrValue.String
		case label.IntValue.Valid:
			value = strconv.Itoa(int(label.IntValue.Int32))
		default:
			continue
		}

		pair := WorkerLabelPair{Key: label.Key, Value: value}

		if _, ok := seen[pair]; ok {
			continue
		}

		seen[pair] = struct{}{}
		pairs = append(pairs, pair)
	}

	return pairs
}

func (p *PrometheusExtension) ReportSnapshot(ctx context.Context, tenantId uuid.UUID, input *SnapshotInput) {
	ctx, span := telemetry.NewSpan(ctx, "prometheus-extension-report-snapshot")
	defer span.End()

	p.mu.Lock()
	defer p.mu.Unlock()

	tenantIdStr := tenantId.String()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantIdStr},
		telemetry.AttributeKV{Key: "snapshot.worker_count", Value: len(input.Workers)},
	)

	tenantMetricsEnabled := p.promGate.Enabled(ctx, tenantId)

	workerPromLabelsToSlotData := make(map[WorkerPromLabels]*SlotUtilization)
	labelPairsToSlotData := make(map[LabelPairPromLabels]*SlotUtilization)

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

		// A worker's slots count towards every (key, value) label pair it carries,
		// so series for different label keys overlap and should not be summed
		// across keys. Slot types are disjoint capacity pools, so summing across
		// slot types is fine.
		for _, pair := range workerLabelPairs(worker.Labels) {
			for slotType, typeUtilization := range input.WorkerSlotUtilizationByType[workerId] {
				seriesLabels := LabelPairPromLabels{
					WorkerLabelPair: pair,
					SlotType:        slotType,
				}

				pairData, ok := labelPairsToSlotData[seriesLabels]
				if ok {
					pairData.UtilizedSlots += typeUtilization.UtilizedSlots
					pairData.NonUtilizedSlots += typeUtilization.NonUtilizedSlots
				} else {
					labelPairsToSlotData[seriesLabels] = &SlotUtilization{
						UtilizedSlots:    typeUtilization.UtilizedSlots,
						NonUtilizedSlots: typeUtilization.NonUtilizedSlots,
					}
				}
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

	if known, ok := p.tenantIdToLabelPairs[tenantId]; ok {
		for pair := range known {
			if _, stillActive := labelPairsToSlotData[pair]; !stillActive {
				prometheus.TenantWorkerLabelSlots.DeleteLabelValues(tenantIdStr, pair.Key, pair.Value, pair.SlotType)
				prometheus.TenantUsedWorkerLabelSlots.DeleteLabelValues(tenantIdStr, pair.Key, pair.Value, pair.SlotType)
				prometheus.TenantAvailableWorkerLabelSlots.DeleteLabelValues(tenantIdStr, pair.Key, pair.Value, pair.SlotType)
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

		if !tenantMetricsEnabled {
			continue
		}

		prometheus.TenantWorkerSlots.WithLabelValues(tenantIdStr, promLabels.ID.String(), promLabels.Name).Set(usedSlots + availableSlots)
		prometheus.TenantUsedWorkerSlots.WithLabelValues(tenantIdStr, promLabels.ID.String(), promLabels.Name).Set(usedSlots)
		prometheus.TenantAvailableWorkerSlots.WithLabelValues(tenantIdStr, promLabels.ID.String(), promLabels.Name).Set(availableSlots)
	}

	currentPairs := make(map[LabelPairPromLabels]struct{}, len(labelPairsToSlotData))
	for pair, utilization := range labelPairsToSlotData {
		currentPairs[pair] = struct{}{}

		usedSlots := float64(utilization.UtilizedSlots)
		availableSlots := float64(utilization.NonUtilizedSlots)

		// Same transient 0-slot rule as the per-worker gauges: writing a 0 total
		// would make utilization queries divide by zero during replenishment gaps.
		if usedSlots+availableSlots == 0 {
			continue
		}

		if !tenantMetricsEnabled {
			continue
		}

		prometheus.TenantWorkerLabelSlots.WithLabelValues(tenantIdStr, pair.Key, pair.Value, pair.SlotType).Set(usedSlots + availableSlots)
		prometheus.TenantUsedWorkerLabelSlots.WithLabelValues(tenantIdStr, pair.Key, pair.Value, pair.SlotType).Set(usedSlots)
		prometheus.TenantAvailableWorkerLabelSlots.WithLabelValues(tenantIdStr, pair.Key, pair.Value, pair.SlotType).Set(availableSlots)
	}

	p.tenantIdToWorkerLabels[tenantId] = currentWorkers
	p.tenantIdToLabelPairs[tenantId] = currentPairs
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

	if known, ok := p.tenantIdToLabelPairs[tenantId]; ok {
		for pair := range known {
			prometheus.TenantWorkerLabelSlots.DeleteLabelValues(tenantIdStr, pair.Key, pair.Value, pair.SlotType)
			prometheus.TenantUsedWorkerLabelSlots.DeleteLabelValues(tenantIdStr, pair.Key, pair.Value, pair.SlotType)
			prometheus.TenantAvailableWorkerLabelSlots.DeleteLabelValues(tenantIdStr, pair.Key, pair.Value, pair.SlotType)
		}

		delete(p.tenantIdToLabelPairs, tenantId)
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

	for tenantId := range p.tenantIdToLabelPairs {
		p.deleteGaugesForTenant(tenantId)
	}

	p.tenants = make(map[uuid.UUID]*sqlcv1.Tenant)
	p.tenantIdToWorkerLabels = make(map[uuid.UUID]map[WorkerPromLabels]struct{})
	p.tenantIdToLabelPairs = make(map[uuid.UUID]map[LabelPairPromLabels]struct{})
	return nil
}
