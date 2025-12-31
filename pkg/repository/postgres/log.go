package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type logAPIRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewLogAPIRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.LogsAPIRepository {
	queries := dbsqlc.New()

	return &logAPIRepository{
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

type logEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

// Used as hook a hook to allow for additional configuration to be passed to the repository if it is instantiated a different way
func (le *logAPIRepository) WithAdditionalConfig(v validator.Validator, l *zerolog.Logger) repository.LogsAPIRepository {
	panic("not implemented in this repo")

}

// Used as hook a hook to allow for additional configuration to be passed to the repository if it is instantiated a different way
func (le *logEngineRepository) WithAdditionalConfig(v validator.Validator, l *zerolog.Logger) repository.LogsEngineRepository {
	panic("not implemented in this repo")
}

func NewLogEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.LogsEngineRepository {
	queries := dbsqlc.New()

	return &logEngineRepository{
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}
