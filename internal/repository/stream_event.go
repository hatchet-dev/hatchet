package repository

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type CreateStreamEventOpts struct {
	// The step run id
	StepRunId string `validate:"required,uuid"`

	// (optional) The time when the StreamEvent line was created.
	CreatedAt *time.Time

	// (required) The message of the StreamEvent line.
	Message byte `validate:"required,min=1,max=10000"` // TODO validation string

	// (optional) The metadata of the StreamEvent line.
	Metadata []byte
}

type ListStreamEventsOpts struct {
	// (optional) number of StreamEvents to skip
	Offset *int

	// (optional) number of StreamEvents to return
	Limit *int `validate:"omitnil,min=1,max=1000"`

	// (optional) a step run id to filter by
	StepRunId *string `validate:"omitempty,uuid"`

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=createdAt"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`
}

type ListStreamEventsResult struct {
	Rows  []*dbsqlc.StreamEvent
	Count int
}

type StreamEventsAPIRepository interface {
	// ListStreamEvents returns a list of StreamEvent lines for a given step run.
	ListStreamEvents(tenantId string, opts *ListStreamEventsOpts) (*ListStreamEventsResult, error)
}

type StreamEventsEngineRepository interface {
	// PutStreamEvent creates a new StreamEvent line.
	PutStreamEvent(tenantId string, opts *CreateStreamEventOpts) (*dbsqlc.StreamEvent, error)
}
