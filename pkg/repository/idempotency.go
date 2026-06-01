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

type KeyClaimantPair struct {
	IdempotencyKey      IdempotencyKey
	ClaimedByExternalId uuid.UUID
}

func claimIdempotencyKeys(context context.Context, queries *sqlcv1.Queries, tx sqlcv1.DBTX, tenantId uuid.UUID, claims []KeyClaimantPair) (map[KeyClaimantPair]WasSuccessfullyClaimed, error) {
	keyToClaimStatus := make(map[KeyClaimantPair]WasSuccessfullyClaimed)

	if len(claims) == 0 {
		return keyToClaimStatus, nil
	}

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
