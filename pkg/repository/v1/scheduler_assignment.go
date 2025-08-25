package v1

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type assignmentRepository struct {
	*sharedRepository
}

func newAssignmentRepository(shared *sharedRepository) *assignmentRepository {
	return &assignmentRepository{
		sharedRepository: shared,
	}
}

func (d *assignmentRepository) ListActionsForWorkers(ctx context.Context, tenantId pgtype.UUID, workerIds []pgtype.UUID) ([]*sqlcv1.ListActionsForWorkersRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-actions-for-workers")
	defer span.End()

	return d.queries.ListActionsForWorkers(ctx, d.pool, sqlcv1.ListActionsForWorkersParams{
		Tenantid:  tenantId,
		Workerids: workerIds,
	})
}

type WorkerSlots struct {
	Used      int32
	Total     int32
	Available int32
}

func (d *assignmentRepository) ListAvailableSlotsForWorkers(ctx context.Context, tenantId pgtype.UUID, params sqlcv1.ListAvailableSlotsForWorkersParams) ([]*sqlcv1.ListAvailableSlotsForWorkersRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-available-slots-for-workers")
	defer span.End()

	slots, err := d.queries.ListAvailableSlotsForWorkers(ctx, d.pool, params)

	if err != nil {
		return nil, err
	}

	workerNameToSlots := make(map[string]*WorkerSlots)

	for _, slot := range slots {
		if data, ok := workerNameToSlots[slot.Name]; ok {
			data.Used += int32(slot.UsedSlots)
			data.Total += slot.TotalSlots
			data.Available += slot.AvailableSlots
		} else {
			workerNameToSlots[slot.Name] = &WorkerSlots{
				Used:      int32(slot.UsedSlots),
				Total:     slot.TotalSlots,
				Available: slot.AvailableSlots,
			}
		}
	}

	for workerName, slotData := range workerNameToSlots {
		prometheus.TenantUsedWorkerSlots.WithLabelValues(workerName).Set(float64(slotData.Total))
		prometheus.TenantUsedWorkerSlots.WithLabelValues(workerName).Set(float64(slotData.Used))
	}

	return slots, nil
}
