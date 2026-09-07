package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type serverlessLeaseRepository struct {
	*sharedRepository
}

func (r *serverlessLeaseRepository) Claim(ctx context.Context, processId uuid.UUID, deadIds []uuid.UUID, limit int32) ([]*sqlcv1.ClaimServerlessLeasesRow, error) {
	if limit <= 0 {
		return nil, nil
	}

	if deadIds == nil {
		deadIds = []uuid.UUID{}
	}

	return r.queries.ClaimServerlessLeases(ctx, r.pool, sqlcv1.ClaimServerlessLeasesParams{
		Processid:  processId,
		Deadids:    deadIds,
		Claimlimit: limit,
	})
}

func (r *serverlessLeaseRepository) Shed(ctx context.Context, processId uuid.UUID, units []ServerlessUnit) ([]*sqlcv1.ShedServerlessLeasesRow, error) {
	if len(units) == 0 {
		return nil, nil
	}

	tenantIds, shards := unitArrays(units)

	return r.queries.ShedServerlessLeases(ctx, r.pool, sqlcv1.ShedServerlessLeasesParams{
		Processid: processId,
		Tenantids: tenantIds,
		Shards:    shards,
	})
}

func (r *serverlessLeaseRepository) ReleaseAll(ctx context.Context, processId uuid.UUID) (int64, error) {
	return r.queries.ReleaseAllServerlessLeases(ctx, r.pool, processId)
}

func (r *serverlessLeaseRepository) ListOwned(ctx context.Context, processId uuid.UUID) ([]*sqlcv1.V1ServerlessLease, error) {
	return r.queries.ListOwnedServerlessLeases(ctx, r.pool, processId)
}

func (r *serverlessLeaseRepository) CountUnowned(ctx context.Context) (*sqlcv1.CountUnownedServerlessLeasesRow, error) {
	return r.queries.CountUnownedServerlessLeases(ctx, r.pool)
}

func (r *serverlessLeaseRepository) InsertIfAbsent(ctx context.Context, unit ServerlessUnit) error {
	return r.queries.InsertServerlessLeaseIfAbsent(ctx, r.pool, sqlcv1.InsertServerlessLeaseIfAbsentParams{
		Tenantid: unit.TenantId,
		Shard:    unit.Shard,
	})
}

func (r *serverlessLeaseRepository) IncrementEndpointCount(ctx context.Context, unit ServerlessUnit, delta int32) error {
	return r.queries.IncrementServerlessLeaseEndpointCount(ctx, r.pool, sqlcv1.IncrementServerlessLeaseEndpointCountParams{
		Tenantid: unit.TenantId,
		Shard:    unit.Shard,
		Delta:    delta,
	})
}
