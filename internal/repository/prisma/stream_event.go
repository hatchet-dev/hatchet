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

type streamEventAPIRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewStreamEventAPIRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.StreamEventsAPIRepository {
	queries := dbsqlc.New()

	return &streamEventAPIRepository{
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

func (r *streamEventAPIRepository) ListStreamEvents(tenantId string, opts *repository.ListStreamEventsOpts) (*repository.ListStreamEventsResult, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	res := &repository.ListStreamEventsResult{}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	queryParams := dbsqlc.ListStreamEventsParams{
		Tenantid: pgTenantId,
	}

	countParams := dbsqlc.CountStreamEventsParams{
		Tenantid: pgTenantId,
	}

	if opts.Offset != nil {
		queryParams.Offset = *opts.Offset
	}

	if opts.Limit != nil {
		queryParams.Limit = *opts.Limit
	}

	if opts.StepRunId != nil {
		queryParams.StepRunId = sqlchelpers.UUIDFromStr(*opts.StepRunId)
		countParams.StepRunId = sqlchelpers.UUIDFromStr(*opts.StepRunId)
	}

	orderByField := "createdAt"
	orderByDirection := "DESC"

	if opts.OrderBy != nil {
		orderByField = *opts.OrderBy
	}

	if opts.OrderDirection != nil {
		orderByDirection = *opts.OrderDirection
	}

	queryParams.OrderBy = sqlchelpers.TextFromStr(orderByField + " " + orderByDirection)

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), r.l, tx.Rollback)

	streamEvents, err := r.queries.ListStreamEvents(context.Background(), tx, queryParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			streamEvents = make([]*dbsqlc.StreamEvent, 0)
		} else {
			return nil, fmt.Errorf("could not list log lines: %w", err)
		}
	}

	count, err := r.queries.CountStreamEvents(context.Background(), tx, countParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			count = 0
		} else {
			return nil, fmt.Errorf("could not count events: %w", err)
		}
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	res.Rows = streamEvents
	res.Count = int(count)

	return res, nil
}

type streamEventEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewStreamEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.StreamEventsEngineRepository {
	queries := dbsqlc.New()

	return &streamEventEngineRepository{
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

func (r *streamEventEngineRepository) PutStreamEvent(tenantId string, opts *repository.CreateStreamEventOpts) (*dbsqlc.StreamEvent, error) {
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

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), r.l, tx.Rollback)

	streamEvent, err := r.queries.CreateStreamEvent(
		context.Background(),
		tx,
		createParams,
	)

	if err != nil {
		return nil, fmt.Errorf("could not create stream devent: %w", err)
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return streamEvent, nil
}
