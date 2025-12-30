package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type getGroupKeyRunRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
}

func NewGetGroupKeyRunRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.GetGroupKeyRunEngineRepository {
	queries := dbsqlc.New()

	return &getGroupKeyRunRepository{
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}
