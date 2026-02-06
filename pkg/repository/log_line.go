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
	Offset           *int
	Limit            *int `validate:"omitnil,min=1,max=10000"`
	Search           *string
	Since            *time.Time
	Until            *time.Time
	Attempt          *int32
	OrderByDirection *string  `validate:"omitempty,oneof=ASC DESC"`
	Levels           []string `validate:"omitnil,dive,oneof=INFO ERROR WARN DEBUG"`
}

type CreateLogLineOpts struct {
	CreatedAt      *time.Time
	Level          *string `validate:"omitnil,oneof=INFO ERROR WARN DEBUG"`
	TaskInsertedAt pgtype.Timestamptz
	Message        string `validate:"required,min=1,max=10000"`
	Metadata       []byte
	TaskId         int64
	RetryCount     int
	TaskExternalId uuid.UUID `validate:"required"`
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
