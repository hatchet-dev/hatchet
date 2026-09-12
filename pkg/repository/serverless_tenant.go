package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type serverlessTenantRepository struct {
	*sharedRepository
}

func (r *serverlessTenantRepository) Upsert(ctx context.Context, tenantId uuid.UUID) (*sqlcv1.V1ServerlessTenant, error) {
	return r.queries.UpsertServerlessTenant(ctx, r.pool, tenantId)
}

func (r *serverlessTenantRepository) Get(ctx context.Context, tenantId uuid.UUID) (*sqlcv1.V1ServerlessTenant, error) {
	return r.queries.GetServerlessTenant(ctx, r.pool, tenantId)
}

func (r *serverlessTenantRepository) UpdateShardCount(ctx context.Context, tenantId uuid.UUID, shardCount int32) (*sqlcv1.V1ServerlessTenant, error) {
	if shardCount < 1 {
		return nil, fmt.Errorf("shard count must be at least 1, got %d", shardCount)
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	if _, err := r.queries.UpsertServerlessTenant(ctx, tx, tenantId); err != nil {
		return nil, fmt.Errorf("could not upsert serverless tenant: %w", err)
	}

	tenant, err := r.queries.UpdateServerlessTenantShardCount(ctx, tx, sqlcv1.UpdateServerlessTenantShardCountParams{
		Tenantid:   tenantId,
		Shardcount: shardCount,
	})

	if err != nil {
		return nil, fmt.Errorf("could not update serverless tenant shard count: %w", err)
	}

	// Every shard needs a lease row so a process can own it as soon as an endpoint lands on it.
	for shard := int32(0); shard < shardCount; shard++ {
		err := r.queries.InsertServerlessLeaseIfAbsent(ctx, tx, sqlcv1.InsertServerlessLeaseIfAbsentParams{
			Tenantid: tenantId,
			Shard:    shard,
		})

		if err != nil {
			return nil, fmt.Errorf("could not create serverless lease unit for shard %d: %w", shard, err)
		}
	}

	if err := commit(ctx); err != nil {
		return nil, fmt.Errorf("could not commit serverless tenant shard count update: %w", err)
	}

	return tenant, nil
}
