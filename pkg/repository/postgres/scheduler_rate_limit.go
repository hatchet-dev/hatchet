package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type rateLimitRepository struct {
	*sharedRepository
}

func newRateLimitRepository(shared *sharedRepository) *rateLimitRepository {
	return &rateLimitRepository{
		sharedRepository: shared,
	}
}

func (d *rateLimitRepository) ListCandidateRateLimits(ctx context.Context, tenantId pgtype.UUID) ([]string, error) {
	rls, err := d.queries.ListRateLimitsForTenantNoMutate(ctx, d.pool, dbsqlc.ListRateLimitsForTenantNoMutateParams{
		Tenantid: tenantId,
		Limit:    10000,
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

func (d *rateLimitRepository) UpdateRateLimits(ctx context.Context, tenantId pgtype.UUID, updates map[string]int) (map[string]int, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	params := dbsqlc.BulkUpdateRateLimitsParams{
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
		return nil, err
	}

	newRls, err := d.queries.ListRateLimitsForTenantWithMutate(ctx, tx, tenantId)

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	res := make(map[string]int, len(newRls))

	for _, rl := range newRls {
		res[rl.Key] = int(rl.Value)
	}

	return res, err
}
