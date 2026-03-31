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

	// (optional) a list of task external ids to filter by
	TaskExternalIds []uuid.UUID
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

	// the workflow id associated with the log line, used for partitioning logs
	WorkflowId uuid.UUID
}

type ListLogLineRow struct {
	*sqlcv1.V1LogLine
	TaskDisplayName string
	TaskExternalId  uuid.UUID
}

type GetLogLinePointMetricsOpts struct {
	StartTimestamp  time.Time `validate:"required"`
	EndTimestamp    time.Time `validate:"required"`
	Search          *string
	Levels          []string `validate:"omitnil,dive,oneof=INFO ERROR WARN DEBUG"`
	TaskExternalIds []uuid.UUID
	BucketInterval  time.Duration `validate:"required"`
}

type LogLineRepository interface {
	ListLogLines(ctx context.Context, tenantId uuid.UUID, opts *ListLogsOpts) ([]*ListLogLineRow, error)

	PutLog(ctx context.Context, tenantId uuid.UUID, opts *CreateLogLineOpts) error

	GetLogLinePointMetrics(ctx context.Context, tenantId uuid.UUID, opts *GetLogLinePointMetricsOpts) ([]*sqlcv1.GetLogLinePointMetricsRow, error)
}

type logLineRepositoryImpl struct {
	*sharedRepository
}

func newLogLineRepository(s *sharedRepository) LogLineRepository {
	return &logLineRepositoryImpl{
		sharedRepository: s,
	}
}

func (r *logLineRepositoryImpl) ListLogLines(ctx context.Context, tenantId uuid.UUID, opts *ListLogsOpts) ([]*ListLogLineRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	queryParams := sqlcv1.ListLogLinesParams{
		Tenantid:         tenantId,
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

	if len(opts.TaskExternalIds) > 0 {
		internalIds, err := r.resolveTaskExternalIds(ctx, tenantId, opts.TaskExternalIds)
		if err != nil {
			return nil, err
		}
		queryParams.TaskIds = internalIds
	}

	logLines, err := r.queries.ListLogLines(ctx, r.pool, queryParams)
	if err != nil {
		return nil, err
	}

	// gather unique task ids, inserted ats to look up associated task external ids
	uniqueTaskIds := make(map[int64]struct{})
	for _, logLine := range logLines {
		uniqueTaskIds[logLine.TaskID] = struct{}{}
	}

	taskIds := make([]int64, 0, len(uniqueTaskIds))
	for taskId := range uniqueTaskIds {
		taskIds = append(taskIds, taskId)
	}

	// look up associated task external ids
	tasks, err := r.listTasks(ctx, r.pool, tenantId, taskIds)

	if err != nil {
		return nil, err
	}

	// create a map of task id to external id
	taskIdToTask := make(map[int64]*sqlcv1.V1Task)
	for _, task := range tasks {
		taskIdToTask[task.ID] = task
	}

	// attach task external ids to log lines
	res := make([]*ListLogLineRow, len(logLines))

	for i, logLine := range logLines {
		task, ok := taskIdToTask[logLine.TaskID]

		if !ok {
			continue
		}

		res[i] = &ListLogLineRow{
			V1LogLine:       logLine,
			TaskExternalId:  task.ExternalID,
			TaskDisplayName: task.DisplayName,
		}
	}

	return res, nil
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

func (r *logLineRepositoryImpl) GetLogLinePointMetrics(ctx context.Context, tenantId uuid.UUID, opts *GetLogLinePointMetricsOpts) ([]*sqlcv1.GetLogLinePointMetricsRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := sqlcv1.GetLogLinePointMetricsParams{
		Interval:      durationToPgInterval(opts.BucketInterval),
		Tenantid:      tenantId,
		Createdafter:  sqlchelpers.TimestamptzFromTime(opts.StartTimestamp),
		Createdbefore: sqlchelpers.TimestamptzFromTime(opts.EndTimestamp),
	}

	if opts.Search != nil {
		params.Search = sqlchelpers.TextFromStr(*opts.Search)
	}

	if len(opts.Levels) > 0 {
		lvls := make([]sqlcv1.V1LogLineLevel, len(opts.Levels))
		for i, l := range opts.Levels {
			lvls[i] = sqlcv1.V1LogLineLevel(l)
		}
		params.Levels = lvls
	}

	if len(opts.TaskExternalIds) > 0 {
		internalIds, err := r.resolveTaskExternalIds(ctx, tenantId, opts.TaskExternalIds)
		if err != nil {
			return nil, err
		}
		params.TaskIds = internalIds
	}

	rows, err := r.queries.GetLogLinePointMetrics(ctx, r.pool, params)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *logLineRepositoryImpl) resolveTaskExternalIds(ctx context.Context, tenantId uuid.UUID, externalIds []uuid.UUID) ([]int64, error) {
	tasks, err := r.queries.FlattenExternalIds(ctx, r.pool, sqlcv1.FlattenExternalIdsParams{
		Tenantid:    tenantId,
		Externalids: externalIds,
	})
	if err != nil {
		return nil, err
	}

	ids := make([]int64, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	return ids, nil
}
