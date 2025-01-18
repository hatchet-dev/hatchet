package repository

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type CreateLogLineOpts struct {
	// The step run id
	StepRunId string `validate:"required,uuid"`

	// (optional) The time when the log line was created.
	CreatedAt *time.Time

	// (required) The message of the log line.
	Message string `validate:"required,min=1,max=10000"`

	// (optional) The level of the log line.
	Level *string `validate:"omitnil,oneof=INFO ERROR WARN DEBUG"`

	// (optional) The metadata of the log line.
	Metadata []byte
}

type ListLogsOpts struct {
	// (optional) number of logs to skip
	Offset *int

	// (optional) number of logs to return
	Limit *int `validate:"omitnil,min=1,max=1000"`

	// (optional) a list of log levels to filter by
	Levels []string `validate:"omitnil,dive,oneof=INFO ERROR WARN DEBUG"`

	// (optional) a step run id to filter by
	StepRunId *string `validate:"omitempty,uuid"`

	// (optional) a search query
	Search *string

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=createdAt"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`
}

type ListLogsResult struct {
	Rows  []*dbsqlc.LogLine
	Count int
}

type LogsAPIRepository interface {
	// ListLogLines returns a list of log lines for a given step run.
	ListLogLines(tenantId string, opts *ListLogsOpts) (*ListLogsResult, error)
	WithAdditionalConfig(validator.Validator, *zerolog.Logger) LogsAPIRepository
}

type LogsEngineRepository interface {
	// PutLog creates a new log line.
	PutLog(ctx context.Context, tenantId string, opts *CreateLogLineOpts) (*dbsqlc.LogLine, error)
	WithAdditionalConfig(validator.Validator, *zerolog.Logger) LogsEngineRepository
}
