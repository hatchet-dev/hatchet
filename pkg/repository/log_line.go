package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type ListLogsOpts struct {
	// (optional) number of logs to skip
	Offset *int

	// (optional) number of logs to return
	Limit *int `validate:"omitnil,min=1,max=10000"`

	// (optional) a list of log levels to filter by
	Levels []string `validate:"omitnil,dive,oneof=INFO ERROR WARN DEBUG"`

	// (optional) a search query
	Search *string

	// (optional) the start time to get logs for
	Since *time.Time

	// (optional) the end time to get logs for
	Until *time.Time

	// (optional) the attempt number to filter for
	Attempt *int32

	// (optional) Order by direction
	OrderByDirection *string `validate:"omitempty,oneof=ASC DESC"`
}

type CreateLogLineOpts struct {
	TaskExternalId uuid.UUID `validate:"required"`

	TaskId int64

	TaskInsertedAt pgtype.Timestamptz

	// (optional) The time when the log line was created.
	CreatedAt *time.Time

	// (required) The message of the log line.
	Message string `validate:"required,min=1,max=10000"`

	// (optional) The level of the log line.
	Level *string `validate:"omitnil,oneof=INFO ERROR WARN DEBUG"`

	// (optional) The metadata of the log line.
	Metadata []byte

	// The retry count of the log line.
	RetryCount int
}

type LogLineRepository interface {
	ListLogLines(ctx context.Context, tenantId, taskExternalId uuid.UUID, opts *ListLogsOpts) ([]*sqlcv1.V1LogLine, error)

	PutLog(ctx context.Context, tenantId uuid.UUID, opts *CreateLogLineOpts) error
}

type logLineRepositoryImpl struct {
	*sharedRepository
}

func newLogLineRepository(s *sharedRepository) LogLineRepository {
	return &logLineRepositoryImpl{
		sharedRepository: s,
	}
}

func (r *logLineRepositoryImpl) ListLogLines(ctx context.Context, tenantId, taskExternalId uuid.UUID, opts *ListLogsOpts) ([]*sqlcv1.V1LogLine, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	// get the task id and inserted at
	task, err := r.GetTaskByExternalId(ctx, tenantId, taskExternalId, false)

	if err != nil {
		return nil, err
	}

	queryParams := sqlcv1.ListLogLinesParams{
		Tenantid:         tenantId,
		Taskid:           task.ID,
		Taskinsertedat:   task.InsertedAt,
		Orderbydirection: "ASC",
	}

	if opts.Search != nil {
		queryParams.Search = sqlchelpers.TextFromStr(*opts.Search)
	}

	if opts.Limit != nil {
		queryParams.Limit = pgtype.Int8{
			Int64: int64(*opts.Limit),
			Valid: true,
		}
	}

	if opts.Offset != nil {
		queryParams.Offset = pgtype.Int8{
			Int64: int64(*opts.Offset),
			Valid: true,
		}
	}

	if opts.Since != nil {
		queryParams.Since = pgtype.Timestamptz{
			Time:  *opts.Since,
			Valid: true,
		}
	}

	if opts.Until != nil {
		queryParams.Until = pgtype.Timestamptz{
			Time:  *opts.Until,
			Valid: true,
		}
	}

	if opts.Levels != nil {
		levels := make([]sqlcv1.V1LogLineLevel, len(opts.Levels))
		for i, level := range opts.Levels {
			levels[i] = sqlcv1.V1LogLineLevel(level)
		}

		queryParams.Levels = levels
	}

	if opts.OrderByDirection != nil {
		queryParams.Orderbydirection = *opts.OrderByDirection
	}

	if opts.Attempt != nil {
		queryParams.Attempt = pgtype.Int4{
			Int32: *opts.Attempt,
			Valid: true,
		}
	}

	logLines, err := r.queries.ListLogLines(ctx, r.pool, queryParams)
	if err != nil {
		return nil, err
	}

	return logLines, nil
}

func (r *logLineRepositoryImpl) PutLog(ctx context.Context, tenantId uuid.UUID, opts *CreateLogLineOpts) error {
	if err := r.v.Validate(opts); err != nil {
		return err
	}

	var level sqlcv1.V1LogLineLevel

	if opts.Level == nil {
		level = sqlcv1.V1LogLineLevel("INFO")
	} else {
		level = sqlcv1.V1LogLineLevel(*opts.Level)
	}

	_, err := r.queries.InsertLogLine(
		ctx,
		r.pool,
		[]sqlcv1.InsertLogLineParams{
			{
				TenantID:       tenantId,
				TaskID:         opts.TaskId,
				TaskInsertedAt: opts.TaskInsertedAt,
				Message:        opts.Message,
				RetryCount:     int32(opts.RetryCount),
				Level:          level,
				Metadata:       opts.Metadata,
			},
		},
	)

	return err
}
