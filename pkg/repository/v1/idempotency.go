package v1

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
)

type IdempotencyRepository interface {
	CreateIdempotencyKey(context context.Context, tenantId, key string, expiresAt pgtype.Timestamptz) (bool, error)
}

type idempotencyRepository struct {
	*sharedRepository
}

func newIdempotencyRepository(shared *sharedRepository) IdempotencyRepository {
	return &idempotencyRepository{
		sharedRepository: shared,
	}
}

func (r *idempotencyRepository) CreateIdempotencyKey(context context.Context, tenantId, key string, expiresAt pgtype.Timestamptz) (bool, error) {
	return r.queries.CreateIdempotencyKey(context, r.pool, sqlcv1.CreateIdempotencyKeyParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Key:       key,
		Expiresat: expiresAt,
	})
}
