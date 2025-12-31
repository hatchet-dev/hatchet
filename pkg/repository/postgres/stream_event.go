package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type streamEventEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewStreamEventsEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.StreamEventsEngineRepository {
	queries := dbsqlc.New()

	return &streamEventEngineRepository{
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}
