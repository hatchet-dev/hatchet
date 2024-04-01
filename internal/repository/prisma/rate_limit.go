package prisma

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
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

func (r *rateLimitEngineRepository) CreateRateLimit(tenantId string, opts *repository.CreateRateLimitOpts) (*dbsqlc.RateLimit, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	createParams := dbsqlc.CreateRateLimitParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Key:      opts.Key,
		Max:      int32(opts.Max),
		Window:   sqlchelpers.TextFromStr(fmt.Sprintf("1 %s", opts.Unit)),
	}

	rateLimit, err := r.queries.CreateRateLimit(context.Background(), r.pool, createParams)

	if err != nil {
		return nil, fmt.Errorf("could not create rate limit: %w", err)
	}

	return rateLimit, nil
}
