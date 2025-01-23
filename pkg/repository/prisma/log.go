package prisma

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
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

func (r *logAPIRepository) ListLogLines(tenantId string, opts *repository.ListLogsOpts) (*repository.ListLogsResult, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	res := &repository.ListLogsResult{}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	queryParams := dbsqlc.ListLogLinesParams{
		Tenantid: pgTenantId,
	}

	countParams := dbsqlc.CountLogLinesParams{
		Tenantid: pgTenantId,
	}

	if opts.Search != nil {
		queryParams.Search = sqlchelpers.TextFromStr(*opts.Search)
		countParams.Search = sqlchelpers.TextFromStr(*opts.Search)
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

	if opts.Levels != nil {
		var levels []dbsqlc.LogLineLevel

		for _, level := range opts.Levels {
			levels = append(levels, dbsqlc.LogLineLevel(level))
		}

		queryParams.Levels = levels
		countParams.Levels = levels
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

	defer sqlchelpers.DeferRollback(context.Background(), r.l, tx.Rollback)

	logLines, err := r.queries.ListLogLines(context.Background(), tx, queryParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logLines = make([]*dbsqlc.LogLine, 0)
		} else {
			return nil, fmt.Errorf("could not list log lines: %w", err)
		}
	}

	count, err := r.queries.CountLogLines(context.Background(), tx, countParams)

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

	res.Rows = logLines
	res.Count = int(count)

	return res, nil
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

func (r *logEngineRepository) PutLog(ctx context.Context, tenantId string, opts *repository.CreateLogLineOpts) (*dbsqlc.LogLine, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	createParams := dbsqlc.CreateLogLineParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Message:   opts.Message,
		Steprunid: sqlchelpers.UUIDFromStr(opts.StepRunId),
	}

	if opts.CreatedAt != nil {
		utcTime := opts.CreatedAt.UTC()
		createParams.CreatedAt = sqlchelpers.TimestampFromTime(utcTime)
	}

	if opts.Level != nil {
		createParams.Level = dbsqlc.NullLogLineLevel{
			LogLineLevel: dbsqlc.LogLineLevel(*opts.Level),
			Valid:        true,
		}
	}

	if opts.Metadata != nil {
		createParams.Metadata = opts.Metadata
	}

	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, r.l, tx.Rollback)

	logLine, err := r.queries.CreateLogLine(
		ctx,
		tx,
		createParams,
	)

	if err != nil {
		return nil, fmt.Errorf("could not create log line: %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return logLine, nil
}
