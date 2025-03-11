package v1

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type ListLogsOpts struct {
	// (optional) number of logs to skip
	Offset *int

	// (optional) number of logs to return
	Limit *int `validate:"omitnil,min=1,max=1000"`

	// (optional) a list of log levels to filter by
	Levels []string `validate:"omitnil,dive,oneof=INFO ERROR WARN DEBUG"`

	// (optional) a search query
	Search *string
}

type CreateLogLineOpts struct {
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
}

type LogLineRepository interface {
	ListLogLines(ctx context.Context, tenantId string, taskId int64, taskInsertedAt pgtype.Timestamptz, opts *ListLogsOpts) ([]*sqlcv1.V1LogLine, error)

	PutLog(ctx context.Context, tenantId string, opts *CreateLogLineOpts) error
}

type logLineRepositoryImpl struct {
	*sharedRepository
}

func newLogLineRepository(s *sharedRepository) LogLineRepository {
	return &logLineRepositoryImpl{
		sharedRepository: s,
	}
}

func (r *logLineRepositoryImpl) ListLogLines(ctx context.Context, tenantId string, taskId int64, taskInsertedAt pgtype.Timestamptz, opts *ListLogsOpts) ([]*sqlcv1.V1LogLine, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	queryParams := sqlcv1.ListLogLinesParams{
		Tenantid:       pgTenantId,
		Taskid:         taskId,
		Taskinsertedat: taskInsertedAt,
	}

	if opts.Search != nil {
		queryParams.Search = sqlchelpers.TextFromStr(*opts.Search)
	}

	return r.queries.ListLogLines(ctx, r.pool, queryParams)
}

func (r *logLineRepositoryImpl) PutLog(ctx context.Context, tenantId string, opts *CreateLogLineOpts) error {
	if err := r.v.Validate(opts); err != nil {
		return err
	}

	_, err := r.queries.InsertLogLine(
		ctx,
		r.pool,
		[]sqlcv1.InsertLogLineParams{
			{
				TenantID:       sqlchelpers.UUIDFromStr(tenantId),
				TaskID:         opts.TaskId,
				TaskInsertedAt: opts.TaskInsertedAt,
				Message:        opts.Message,
			},
		},
	)

	return err
}
