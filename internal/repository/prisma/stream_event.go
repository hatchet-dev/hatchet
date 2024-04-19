package prisma

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
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

func (r *streamEventEngineRepository) PutStreamEvent(ctx context.Context, tenantId string, opts *repository.CreateStreamEventOpts) (*dbsqlc.StreamEvent, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	createParams := dbsqlc.CreateStreamEventParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Message:   opts.Message,
		Steprunid: sqlchelpers.UUIDFromStr(opts.StepRunId),
	}

	if opts.CreatedAt != nil {
		utcTime := opts.CreatedAt.UTC()
		createParams.CreatedAt = sqlchelpers.TimestampFromTime(utcTime)
	}

	if opts.Metadata != nil {
		createParams.Metadata = opts.Metadata
	}

	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, r.l, tx.Rollback)

	streamEvent, err := r.queries.CreateStreamEvent(
		ctx,
		tx,
		createParams,
	)

	if err != nil {
		return nil, fmt.Errorf("could not create stream event: %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return streamEvent, nil
}

func (r *streamEventEngineRepository) GetStreamEvent(ctx context.Context, tenantId string, streamEventId int64) (*dbsqlc.StreamEvent, error) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer deferRollback(ctx, r.l, tx.Rollback)

	streamEvent, err := r.queries.GetStreamEvent(ctx, tx, dbsqlc.GetStreamEventParams{
		ID:       streamEventId,
		Tenantid: pgTenantId,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("stream event not found")
		}
		return nil, fmt.Errorf("could not get stream event: %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return streamEvent, nil
}

func (r *streamEventEngineRepository) CleanupStreamEvents(ctx context.Context) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer deferRollback(ctx, r.l, tx.Rollback)

	err = r.queries.CleanupStreamEvents(ctx, r.pool)

	if err != nil {
		return fmt.Errorf("could not cleanup stream events: %w", err)
	}

	return nil
}
