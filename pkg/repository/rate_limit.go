package repository

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type ListRateLimitOpts struct {
	// (optional) a search query for the key
	Search *string

	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=key value limitValue"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`
}

type ListRateLimitsResult struct {
	Rows  []*sqlcv1.ListRateLimitsForTenantNoMutateRow
	Count int
}

type UpsertRateLimitOpts struct {
	// The rate limit max value
	Limit int

	// The rate limit duration
	Duration *string `validate:"omitnil,oneof=SECOND MINUTE HOUR DAY WEEK MONTH YEAR"`
}

type RateLimitRepository interface {
	UpdateRateLimits(ctx context.Context, tenantId uuid.UUID, updates map[string]int) ([]*sqlcv1.ListRateLimitsForTenantWithMutateRow, *time.Time, error)

	UpsertRateLimit(ctx context.Context, tenantId string, key string, opts *UpsertRateLimitOpts) (*sqlcv1.RateLimit, error)

	ListRateLimits(ctx context.Context, tenantId string, opts *ListRateLimitOpts) (*ListRateLimitsResult, error)
}

const MAX_TENANT_RATE_LIMITS = 10000

type rateLimitRepository struct {
	*sharedRepository
}

func newRateLimitRepository(shared *sharedRepository) *rateLimitRepository {
	return &rateLimitRepository{
		sharedRepository: shared,
	}
}

func (r *rateLimitRepository) UpdateRateLimits(ctx context.Context, tenantId uuid.UUID, updates map[string]int) ([]*sqlcv1.ListRateLimitsForTenantWithMutateRow, *time.Time, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

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

	tenantInt := tenantAdvisoryInt(sqlchelpers.UUIDToStr(tenantId))

	err = r.queries.AdvisoryLock(ctx, tx, tenantInt)

	if err != nil {
		return nil, nil, err
	}

	_, err = r.queries.BulkUpdateRateLimits(ctx, tx, params)

	if err != nil {
		return nil, nil, err
	}

	newRls, err := r.queries.ListRateLimitsForTenantWithMutate(ctx, tx, tenantId)

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

	return newRls, &nextRefillAt, err
}

func (r *rateLimitRepository) UpsertRateLimit(ctx context.Context, tenantId string, key string, opts *UpsertRateLimitOpts) (*sqlcv1.RateLimit, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	upsertParams := sqlcv1.UpsertRateLimitParams{
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

func (r *rateLimitRepository) ListRateLimits(ctx context.Context, tenantId string, opts *ListRateLimitOpts) (*ListRateLimitsResult, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	res := &ListRateLimitsResult{}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	queryParams := sqlcv1.ListRateLimitsForTenantNoMutateParams{
		Tenantid: pgTenantId,
	}

	countParams := sqlcv1.CountRateLimitsParams{
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
			rls = make([]*sqlcv1.ListRateLimitsForTenantNoMutateRow, 0)
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

func tenantAdvisoryInt(tenantID string) int64 {
	hasher := fnv.New64a()
	idBytes := []byte(tenantID)
	hasher.Write(idBytes)
	return int64(hasher.Sum64()) // nolint: gosec
}
