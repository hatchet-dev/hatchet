package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type WasSuccessfullyClaimed bool
type IdempotencyKey string

type IdempotencyRepository interface {
	CreateIdempotencyKey(context context.Context, tenantId, key string, expiresAt pgtype.Timestamptz) error
	EvictExpiredIdempotencyKeys(context context.Context, tenantId uuid.UUID) error
}

type idempotencyRepository struct {
	*sharedRepository
}

func newIdempotencyRepository(shared *sharedRepository) IdempotencyRepository {
	return &idempotencyRepository{
		sharedRepository: shared,
	}
}

func (r *idempotencyRepository) CreateIdempotencyKey(context context.Context, tenantId, key string, expiresAt pgtype.Timestamptz) error {
	return r.queries.CreateIdempotencyKey(context, r.pool, sqlcv1.CreateIdempotencyKeyParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Key:       key,
		Expiresat: expiresAt,
	})
}

func (r *idempotencyRepository) EvictExpiredIdempotencyKeys(context context.Context, tenantId uuid.UUID) error {
	return r.queries.CleanUpExpiredIdempotencyKeys(context, r.pool, tenantId)
}

type KeyClaimantPair struct {
	IdempotencyKey      IdempotencyKey
	ClaimedByExternalId uuid.UUID
}

func claimIdempotencyKeys(context context.Context, queries *sqlcv1.Queries, pool *pgxpool.Pool, tenantId string, claims []KeyClaimantPair) (map[KeyClaimantPair]WasSuccessfullyClaimed, error) {
	keys := make([]string, len(claims))
	claimedByExternalIds := make([]uuid.UUID, len(claims))

	for i, claim := range claims {
		keys[i] = string(claim.IdempotencyKey)
		claimedByExternalIds[i] = claim.ClaimedByExternalId
	}

	claimResults, err := queries.ClaimIdempotencyKeys(context, pool, sqlcv1.ClaimIdempotencyKeysParams{
		Tenantid:             sqlchelpers.UUIDFromStr(tenantId),
		Keys:                 keys,
		Claimedbyexternalids: claimedByExternalIds,
	})

	if err != nil {
		return nil, err
	}

	keyToClaimStatus := make(map[KeyClaimantPair]WasSuccessfullyClaimed)

	for _, claimResult := range claimResults {
		keyClaimantPair := KeyClaimantPair{
			IdempotencyKey:      IdempotencyKey(claimResult.Key),
			ClaimedByExternalId: claimResult.ClaimedByExternalID,
		}
		keyToClaimStatus[keyClaimantPair] = WasSuccessfullyClaimed(claimResult.WasSuccessfullyClaimed)
	}

	return keyToClaimStatus, nil
}
