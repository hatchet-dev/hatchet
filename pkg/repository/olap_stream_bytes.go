package repository

import (
	"bytes"
	"context"
	"slices"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// TenantStreamBytes is a tenant's durable stream payload bytes since the last flush.
type TenantStreamBytes struct {
	TenantID       uuid.UUID
	PublishedBytes int64
	DeliveredBytes int64
}

// AddStreamBytes adds each tenant's bytes to the current minute's totals.
func (r *OLAPRepositoryImpl) AddStreamBytes(ctx context.Context, usage []TenantStreamBytes) error {
	usage = slices.DeleteFunc(slices.Clone(usage), func(u TenantStreamBytes) bool {
		return u.PublishedBytes <= 0 && u.DeliveredBytes <= 0
	})

	if len(usage) == 0 {
		return nil
	}

	// a consistent row order keeps concurrent flushes from deadlocking on each other's rows
	slices.SortFunc(usage, func(a, b TenantStreamBytes) int {
		return bytes.Compare(a.TenantID[:], b.TenantID[:])
	})

	params := sqlcv1.AddStreamBytesParams{
		Tenantids:      make([]uuid.UUID, len(usage)),
		Publishedbytes: make([]int64, len(usage)),
		Deliveredbytes: make([]int64, len(usage)),
	}

	for i, u := range usage {
		params.Tenantids[i] = u.TenantID
		params.Publishedbytes[i] = max(u.PublishedBytes, 0)
		params.Deliveredbytes[i] = max(u.DeliveredBytes, 0)
	}

	// shared, so the write doesn't count against the tenants' own database budgets
	return r.queries.AddStreamBytes(ctx, r.pool.ForShared(), params)
}
