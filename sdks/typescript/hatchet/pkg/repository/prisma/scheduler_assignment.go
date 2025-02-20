package prisma

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type assignmentRepository struct {
	*sharedRepository
}

func newAssignmentRepository(shared *sharedRepository) *assignmentRepository {
	return &assignmentRepository{
		sharedRepository: shared,
	}
}

func (d *assignmentRepository) ListActionsForWorkers(ctx context.Context, tenantId pgtype.UUID, workerIds []pgtype.UUID) ([]*dbsqlc.ListActionsForWorkersRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-actions-for-workers")
	defer span.End()

	return d.queries.ListActionsForWorkers(ctx, d.pool, dbsqlc.ListActionsForWorkersParams{
		Tenantid:  tenantId,
		Workerids: workerIds,
	})
}

func (d *assignmentRepository) ListAvailableSlotsForWorkers(ctx context.Context, tenantId pgtype.UUID, params dbsqlc.ListAvailableSlotsForWorkersParams) ([]*dbsqlc.ListAvailableSlotsForWorkersRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-available-slots-for-workers")
	defer span.End()

	return d.queries.ListAvailableSlotsForWorkers(ctx, d.pool, params)
}
