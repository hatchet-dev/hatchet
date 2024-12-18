package prisma

import (
	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type sharedRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries

	bulkStatusBuffer      *buffer.TenantBufferManager[*updateStepRunQueueData, pgtype.UUID]
	bulkEventBuffer       *buffer.TenantBufferManager[*repository.CreateStepRunEventOpts, int]
	bulkSemaphoreReleaser *buffer.TenantBufferManager[semaphoreReleaseOpts, pgtype.UUID]
	bulkQueuer            *buffer.TenantBufferManager[bulkQueueStepRunOpts, pgtype.UUID]
	bulkUserEventBuffer   *buffer.TenantBufferManager[*repository.CreateEventOpts, dbsqlc.Event]
	bulkWorkflowRunBuffer *buffer.TenantBufferManager[*repository.CreateWorkflowRunOpts, dbsqlc.WorkflowRun]
	bulkAckMQBuffer       *buffer.TenantBufferManager[int64, int]
	bulkAddMQBuffer       *buffer.TenantBufferManager[addMessage, int]

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

	statusBuffer, err := newBulkStepRunStatusBuffer(s)

	if err != nil {
		return nil, nil, err
	}

	eventBuffer, err := newBulkEventWriter(s, cf.EventBuffer)

	if err != nil {
		return nil, nil, err
	}

	semaphoreReleaser, err := NewBulkSemaphoreReleaser(s, cf.ReleaseSemaphoreBuffer)

	if err != nil {
		return nil, nil, err
	}

	queuer, err := newBulkStepRunQueuer(s, cf.QueueStepRunBuffer)

	if err != nil {
		return nil, nil, err
	}

	userEventBuffer, err := newUserEventBuffer(s, cf.EventBuffer)

	if err != nil {
		return nil, nil, err
	}

	workflowRunBuffer, err := newCreateWorkflowRunBuffer(s, cf.WorkflowRunBuffer)

	if err != nil {
		return nil, nil, err
	}

	ackMQBuffer, err := newAckMQBuffer(s)

	if err != nil {
		return nil, nil, err
	}

	addMQBuffer, err := newAddMQBuffer(s)

	if err != nil {
		return nil, nil, err
	}

	s.bulkStatusBuffer = statusBuffer
	s.bulkEventBuffer = eventBuffer
	s.bulkSemaphoreReleaser = semaphoreReleaser
	s.bulkQueuer = queuer
	s.bulkUserEventBuffer = userEventBuffer
	s.bulkWorkflowRunBuffer = workflowRunBuffer
	s.bulkAckMQBuffer = ackMQBuffer
	s.bulkAddMQBuffer = addMQBuffer

	return s, func() error {
		var multiErr error

		err := statusBuffer.Cleanup()

		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}

		err = eventBuffer.Cleanup()

		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}

		err = semaphoreReleaser.Cleanup()

		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}

		err = queuer.Cleanup()

		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}

		err = userEventBuffer.Cleanup()

		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}

		err = workflowRunBuffer.Cleanup()

		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}

		err = ackMQBuffer.Cleanup()

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
