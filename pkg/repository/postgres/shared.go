package postgres

import (
	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type sharedRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries

	bulkAckMQBuffer *buffer.TenantBufferManager[int64, int]
	bulkAddMQBuffer *buffer.TenantBufferManager[addMessage, int]

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

	ackMQBuffer, err := newAckMQBuffer(s)

	if err != nil {
		return nil, nil, err
	}

	addMQBuffer, err := newAddMQBuffer(s)

	if err != nil {
		return nil, nil, err
	}

	s.bulkAckMQBuffer = ackMQBuffer
	s.bulkAddMQBuffer = addMQBuffer

	return s, func() error {
		var multiErr error

		err := ackMQBuffer.Cleanup()

		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}

		err = addMQBuffer.Cleanup()

		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}

		return multiErr
	}, nil
}
