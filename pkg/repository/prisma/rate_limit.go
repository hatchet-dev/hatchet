package prisma

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type rateLimitEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewRateLimitEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.RateLimitEngineRepository {
	queries := dbsqlc.New()

	return &rateLimitEngineRepository{
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

func (r *rateLimitEngineRepository) ListRateLimits(ctx context.Context, tenantId string, opts *repository.ListRateLimitOpts) (*repository.ListRateLimitsResult, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	res := &repository.ListRateLimitsResult{}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	queryParams := dbsqlc.ListRateLimitsForTenantNoMutateParams{
		Tenantid: pgTenantId,
	}

	countParams := dbsqlc.CountRateLimitsParams{
		Tenantid: pgTenantId,
	}

	if opts.Search != nil {
		queryParams.Search = sqlchelpers.TextFromStr(*opts.Search)
		countParams.Search = sqlchelpers.TextFromStr(*opts.Search)
	}

	if opts.Offset != nil {
		queryParams.Offset = *opts.Offset
	}

	if opts.Limit != nil {
		queryParams.Limit = *opts.Limit
	}

	orderByField := "key"
	orderByDirection := "ASC"

	if opts.OrderBy != nil {
		orderByField = *opts.OrderBy
	}

	if opts.OrderDirection != nil {
		orderByDirection = *opts.OrderDirection
	}

	queryParams.Orderby = orderByField + " " + orderByDirection
	countParams.Orderby = orderByField + " " + orderByDirection

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(context.Background(), r.l, tx.Rollback)

	rls, err := r.queries.ListRateLimitsForTenantNoMutate(ctx, tx, queryParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			rls = make([]*dbsqlc.ListRateLimitsForTenantNoMutateRow, 0)
		} else {
			return nil, fmt.Errorf("could not list rate limits: %w", err)
		}
	}

	count, err := r.queries.CountRateLimits(ctx, tx, countParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			count = 0
		} else {
			return nil, fmt.Errorf("could not count events: %w", err)
		}
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	res.Rows = rls
	res.Count = int(count)

	return res, nil
}

func (r *rateLimitEngineRepository) UpsertRateLimit(ctx context.Context, tenantId string, key string, opts *repository.UpsertRateLimitOpts) (*dbsqlc.RateLimit, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	upsertParams := dbsqlc.UpsertRateLimitParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Key:      key,
		Limit:    int32(opts.Limit), // nolint: gosec
	}

	if opts.Duration != nil {
		upsertParams.Window = sqlchelpers.TextFromStr(getWindowParamFromDurString(*opts.Duration))
	}

	rateLimit, err := r.queries.UpsertRateLimit(ctx, r.pool, upsertParams)

	if err != nil {
		return nil, fmt.Errorf("could not upsert rate limit: %w", err)
	}

	return rateLimit, nil
}

var durationStrings = []string{
	"SECOND",
	"MINUTE",
	"HOUR",
	"DAY",
	"WEEK",
	"MONTH",
	"YEAR",
}

func getWindowParamFromDurString(dur string) string {
	// validate duration string
	found := false

	for _, d := range durationStrings {
		if d == dur {
			found = true
			break
		}
	}

	if !found {
		return "MINUTE"
	}

	return fmt.Sprintf("1 %s", dur)
}

func getLargerDuration(s1, s2 string) (string, error) {
	i1, err := getDurationIndex(s1)
	if err != nil {
		return "", err
	}

	i2, err := getDurationIndex(s2)
	if err != nil {
		return "", err
	}

	if i1 > i2 {
		return s1, nil
	}

	return s2, nil
}

func getDurationIndex(s string) (int, error) {
	for i, d := range durationStrings {
		if d == s {
			return i, nil
		}
	}

	return -1, fmt.Errorf("invalid duration string: %s", s)
}
