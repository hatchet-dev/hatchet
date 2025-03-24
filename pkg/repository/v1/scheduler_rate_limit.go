package v1

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
)

const MAX_TENANT_RATE_LIMITS = 10000

type rateLimitRepository struct {
	*sharedRepository
}

func newRateLimitRepository(shared *sharedRepository) *rateLimitRepository {
	return &rateLimitRepository{
		sharedRepository: shared,
	}
}

func (d *rateLimitRepository) ListCandidateRateLimits(ctx context.Context, tenantId pgtype.UUID) ([]string, error) {
	rls, err := d.queries.ListRateLimitsForTenantNoMutate(ctx, d.pool, sqlcv1.ListRateLimitsForTenantNoMutateParams{
		Tenantid: tenantId,
		Limit:    MAX_TENANT_RATE_LIMITS,
	})

	if err != nil {
		return nil, err
	}

	ids := make([]string, len(rls))

	for i, rl := range rls {
		ids[i] = rl.Key
	}

	return ids, nil
}

func (d *rateLimitRepository) UpdateRateLimits(ctx context.Context, tenantId pgtype.UUID, updates map[string]int) (map[string]int, *time.Time, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l, 5000)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	params := sqlcv1.BulkUpdateRateLimitsParams{
		Tenantid: tenantId,
		Keys:     make([]string, 0, len(updates)),
		Units:    make([]int32, 0, len(updates)),
	}

	for k, v := range updates {
		params.Keys = append(params.Keys, k)
		params.Units = append(params.Units, int32(v)) // nolint: gosec
	}

	_, err = d.queries.BulkUpdateRateLimits(ctx, tx, params)

	if err != nil {
		return nil, nil, err
	}

	newRls, err := d.queries.ListRateLimitsForTenantWithMutate(ctx, tx, tenantId)

	if err != nil {
		return nil, nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	res := make(map[string]int, len(newRls))

	for _, rl := range newRls {
		res[rl.Key] = int(rl.Value)
	}

	nextRefillAt := time.Now().Add(time.Second * 2)

	if len(newRls) > 0 {
		// get min of all next refill times
		for _, rl := range newRls {
			if rl.NextRefillAt.Time.Before(nextRefillAt) {
				nextRefillAt = rl.NextRefillAt.Time
			}
		}
	}

	return res, &nextRefillAt, err
}
