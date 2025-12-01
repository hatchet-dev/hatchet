package postgres

import (
	"github.com/google/uuid"

	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type assignmentRepository struct {
	*sharedRepository
}

func newAssignmentRepository(shared *sharedRepository) *assignmentRepository {
	return &assignmentRepository{
		sharedRepository: shared,
	}
}

func (d *assignmentRepository) ListActionsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) ([]*dbsqlc.ListActionsForWorkersRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-actions-for-workers")
	defer span.End()

	return d.queries.ListActionsForWorkers(ctx, d.pool, dbsqlc.ListActionsForWorkersParams{
		Tenantid:  tenantId,
		Workerids: workerIds,
	})
}

func (d *assignmentRepository) ListAvailableSlotsForWorkers(ctx context.Context, tenantId uuid.UUID, params dbsqlc.ListAvailableSlotsForWorkersParams) ([]*dbsqlc.ListAvailableSlotsForWorkersRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-available-slots-for-workers")
	defer span.End()

	return d.queries.ListAvailableSlotsForWorkers(ctx, d.pool, params)
}
