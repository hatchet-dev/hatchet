package v1

import (
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	celgo "github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
)

type sharedRepository struct {
	pool       *pgxpool.Pool
	v          validator.Validator
	l          *zerolog.Logger
	queries    *sqlcv1.Queries
	queueCache *cache.Cache
	celParser  *cel.CELParser
	env        *celgo.Env
}

func newSharedRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) (*sharedRepository, func() error) {
	queries := sqlcv1.New()
	cache := cache.New(5 * time.Minute)

	celParser := cel.NewCELParser()

	env, err := celgo.NewEnv(
		celgo.Declarations(
			decls.NewVar("input", decls.NewMapType(decls.String, decls.Dyn)),
		),
	)

	if err != nil {
		log.Fatalf("failed to create CEL environment: %v", err)
	}

	return &sharedRepository{
			pool:       pool,
			v:          v,
			l:          l,
			queries:    queries,
			queueCache: cache,
			celParser:  celParser,
			env:        env,
		}, func() error {
			cache.Stop()
			return nil
		}
}
