package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type WasSuccessfullyClaimed bool
type IdempotencyKey string

type ClaimIdempotencyKeysOpt struct {
	Key                 string
	ClaimedByExternalId uuid.UUID
	ExpiresAt           pgtype.Timestamptz
}

type IdempotencyRepository interface {
	EvictExpiredIdempotencyKeys(context context.Context, tenantId uuid.UUID) error
	ClaimKey(ctx context.Context, tenantId uuid.UUID, key string, expiresAt pgtype.Timestamptz, claimedByExternalId uuid.UUID) (bool, error)
}

type idempotencyRepository struct {
	*sharedRepository
}

func newIdempotencyRepository(shared *sharedRepository) IdempotencyRepository {
	return &idempotencyRepository{
		sharedRepository: shared,
	}
}

func (r *idempotencyRepository) EvictExpiredIdempotencyKeys(context context.Context, tenantId uuid.UUID) error {
	return r.queries.CleanUpExpiredIdempotencyKeys(context, r.pool, tenantId)
}

func (r *idempotencyRepository) ClaimKey(ctx context.Context, tenantId uuid.UUID, key string, expiresAt pgtype.Timestamptz, claimedByExternalId uuid.UUID) (bool, error) {
	results, err := r.queries.ClaimIdempotencyKeys(ctx, r.pool, sqlcv1.ClaimIdempotencyKeysParams{
		Keys:                 []string{key},
		Expiresats:           []pgtype.Timestamptz{expiresAt},
		Claimedbyexternalids: []uuid.UUID{claimedByExternalId},
		Tenantid:             tenantId,
	})
	if err != nil {
		return false, err
	}
	if len(results) == 0 {
		return false, nil
	}
	return results[0].WasSuccessfullyClaimed, nil
}

type IdempotencyCollision struct {
	RequestedExternalId uuid.UUID
	ExistingExternalId  uuid.UUID
}
