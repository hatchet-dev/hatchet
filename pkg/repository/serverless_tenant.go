package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// defaultServerlessEndpointPartitionCount is the endpoint_partition_count a tenant row is created
// with when the caller passes none.
const defaultServerlessEndpointPartitionCount int32 = 1

type serverlessTenantRepository struct {
	*sharedRepository
}

func (r *serverlessTenantRepository) Upsert(ctx context.Context, tenantId uuid.UUID, endpointPartitionCount int32) (*sqlcv1.V1ServerlessTenant, error) {
	return r.queries.UpsertServerlessTenant(ctx, r.pool, upsertServerlessTenantParams(tenantId, endpointPartitionCount))
}

func (r *serverlessTenantRepository) Get(ctx context.Context, tenantId uuid.UUID) (*sqlcv1.V1ServerlessTenant, error) {
	return r.queries.GetServerlessTenant(ctx, r.pool, tenantId)
}

// upsertServerlessTenantParams substitutes the default for a zero partition count.
func upsertServerlessTenantParams(tenantId uuid.UUID, endpointPartitionCount int32) sqlcv1.UpsertServerlessTenantParams {
	if endpointPartitionCount <= 0 {
		endpointPartitionCount = defaultServerlessEndpointPartitionCount
	}

	return sqlcv1.UpsertServerlessTenantParams{Tenantid: tenantId, Endpointpartitioncount: endpointPartitionCount}
}
