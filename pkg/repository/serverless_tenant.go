package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// defaultServerlessShardCount is the shard_count a tenant row is created with when the caller
// passes none.
const defaultServerlessShardCount int32 = 1

type serverlessTenantRepository struct {
	*sharedRepository
}

func (r *serverlessTenantRepository) Upsert(ctx context.Context, tenantId uuid.UUID, shardCount int32) (*sqlcv1.V1ServerlessTenant, error) {
	return r.queries.UpsertServerlessTenant(ctx, r.pool, upsertServerlessTenantParams(tenantId, shardCount))
}

func (r *serverlessTenantRepository) Get(ctx context.Context, tenantId uuid.UUID) (*sqlcv1.V1ServerlessTenant, error) {
	return r.queries.GetServerlessTenant(ctx, r.pool, tenantId)
}

// upsertServerlessTenantParams substitutes the default for a zero shard count.
func upsertServerlessTenantParams(tenantId uuid.UUID, shardCount int32) sqlcv1.UpsertServerlessTenantParams {
	if shardCount <= 0 {
		shardCount = defaultServerlessShardCount
	}

	return sqlcv1.UpsertServerlessTenantParams{Tenantid: tenantId, Shardcount: shardCount}
}
