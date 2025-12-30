package postgres

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type sharedRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries

	wrRunningCallbacks []repository.TenantScopedCallback[pgtype.UUID]
}

func newSharedRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, cf *server.ConfigFileRuntime) (*sharedRepository, func() error, error) {
	queries := dbsqlc.New()

	s := &sharedRepository{
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}

	return s, func() error {
		var multiErr error

		return multiErr
	}, nil
}
