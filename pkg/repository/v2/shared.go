package v2

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type sharedRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *sqlcv2.Queries
}

func newSharedRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) *sharedRepository {
	queries := sqlcv2.New()

	return &sharedRepository{
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}
