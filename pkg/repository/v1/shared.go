package v1

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type sharedRepository struct {
	pool       *pgxpool.Pool
	v          validator.Validator
	l          *zerolog.Logger
	queries    *sqlcv1.Queries
	queueCache *cache.Cache
	celParser  *cel.CELParser
}

func newSharedRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) (*sharedRepository, func() error) {
	queries := sqlcv1.New()
	cache := cache.New(5 * time.Minute)

	celParser := cel.NewCELParser()

	return &sharedRepository{
			pool:       pool,
			v:          v,
			l:          l,
			queries:    queries,
			queueCache: cache,
			celParser:  celParser,
		}, func() error {
			cache.Stop()
			return nil
		}
}
