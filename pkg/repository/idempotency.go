package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type WasSuccessfullyClaimed bool
type IdempotencyKey string

type IdempotencyRepository interface {
	CreateIdempotencyKey(context context.Context, tenantId uuid.UUID, key string, expiresAt pgtype.Timestamptz) error
	CreateIdempotencyKeys(context context.Context, tenantId uuid.UUID, keys []string, expiresAt pgtype.Timestamptz) error
	DeleteIdempotencyKeysByExternalId(context context.Context, tenantId uuid.UUID, externalId uuid.UUID) error
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

func (r *idempotencyRepository) CreateIdempotencyKey(context context.Context, tenantId uuid.UUID, key string, expiresAt pgtype.Timestamptz) error {
	return r.queries.CreateIdempotencyKey(context, r.pool, sqlcv1.CreateIdempotencyKeyParams{
		Tenantid:  tenantId,
		Key:       key,
		Expiresat: expiresAt,
	})
}

func (r *idempotencyRepository) CreateIdempotencyKeys(context context.Context, tenantId uuid.UUID, keys []string, expiresAt pgtype.Timestamptz) error {
	if len(keys) == 0 {
		return nil
	}

	return r.queries.CreateIdempotencyKeys(context, r.pool, sqlcv1.CreateIdempotencyKeysParams{
		Tenantid:  tenantId,
		Keys:      keys,
		Expiresat: expiresAt,
	})
}

func (r *idempotencyRepository) DeleteIdempotencyKeysByExternalId(context context.Context, tenantId uuid.UUID, externalId uuid.UUID) error {
	return r.queries.DeleteIdempotencyKeysByExternalId(context, r.pool, sqlcv1.DeleteIdempotencyKeysByExternalIdParams{
		Tenantid:   tenantId,
		Externalid: externalId,
	})
}

func (r *idempotencyRepository) EvictExpiredIdempotencyKeys(context context.Context, tenantId uuid.UUID) error {
	return r.queries.CleanUpExpiredIdempotencyKeys(context, r.pool, tenantId)
}

type KeyClaimantPair struct {
	IdempotencyKey      IdempotencyKey
	ClaimedByExternalId uuid.UUID
}

func claimIdempotencyKeys(context context.Context, queries *sqlcv1.Queries, tx sqlcv1.DBTX, tenantId uuid.UUID, claims []KeyClaimantPair) (map[KeyClaimantPair]WasSuccessfullyClaimed, error) {
	keys := make([]string, len(claims))
	claimedByExternalIds := make([]uuid.UUID, len(claims))

	for i, claim := range claims {
		keys[i] = string(claim.IdempotencyKey)
		claimedByExternalIds[i] = claim.ClaimedByExternalId
	}

	claimResults, err := queries.ClaimIdempotencyKeys(context, tx, sqlcv1.ClaimIdempotencyKeysParams{
		Tenantid:             tenantId,
		Keys:                 keys,
		Claimedbyexternalids: claimedByExternalIds,
	})

	if err != nil {
		return nil, err
	}

	keyToClaimStatus := make(map[KeyClaimantPair]WasSuccessfullyClaimed)

	for _, claimResult := range claimResults {
		if claimResult.ClaimedByExternalID == nil {
			continue
		}

		keyClaimantPair := KeyClaimantPair{
			IdempotencyKey:      IdempotencyKey(claimResult.Key),
			ClaimedByExternalId: *claimResult.ClaimedByExternalID,
		}
		keyToClaimStatus[keyClaimantPair] = WasSuccessfullyClaimed(claimResult.WasSuccessfullyClaimed)
	}

	return keyToClaimStatus, nil
}
