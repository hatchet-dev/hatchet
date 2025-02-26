package postgres

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type stepRepository struct {
	v       validator.Validator
	pool    *pgxpool.Pool
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewStepRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.StepRepository {
	queries := dbsqlc.New()

	return &stepRepository{
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}

func (j *stepRepository) ListStepExpressions(ctx context.Context, stepId string) ([]*dbsqlc.StepExpression, error) {
	return j.queries.GetStepExpressions(ctx, j.pool, sqlchelpers.UUIDFromStr(stepId))
}
