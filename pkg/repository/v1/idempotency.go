package v1

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
)

type IdempotencyRepository interface {
	CreateIdempotencyKey(context context.Context, tenantId, key string, expiresAt pgtype.Timestamptz) (*sqlcv1.V1IdempotencyKey, error)
	MarkIdempotencyKeyFilled(context context.Context, tenantId, key string) error
	CheckIfIdempotencyKeyFilled(context context.Context, tenantId, key string) (bool, error)
}

type idempotencyRepository struct {
	*sharedRepository
}

func newIdempotencyRepository(shared *sharedRepository) IdempotencyRepository {
	return &idempotencyRepository{
		sharedRepository: shared,
	}
}

func (r *idempotencyRepository) CreateIdempotencyKey(context context.Context, tenantId, key string, expiresAt pgtype.Timestamptz) (*sqlcv1.V1IdempotencyKey, error) {
	return r.queries.CreateIdempotencyKey(context, r.pool, sqlcv1.CreateIdempotencyKeyParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Key:       key,
		Expiresat: expiresAt,
	})
}

func (r *idempotencyRepository) MarkIdempotencyKeyFilled(context context.Context, tenantId, key string) error {
	return r.queries.MarkIdempotencyKeyFilled(context, r.pool, sqlcv1.MarkIdempotencyKeyFilledParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Key:      key,
	})
}

func (r *idempotencyRepository) CheckIfIdempotencyKeyFilled(context context.Context, tenantId, key string) (bool, error) {
	return r.queries.CheckIfIdempotencyKeyFilled(context, r.pool, sqlcv1.CheckIfIdempotencyKeyFilledParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Key:      key,
	})
}
