package repository

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type CreateStreamEventOpts struct {
	// The step run id
	StepRunId string `validate:"required,uuid"`

	// (optional) The time when the StreamEvent was created.
	CreatedAt *time.Time

	// (required) The message of the Stream Event.
	Message []byte `validate:"required,min=1"`

	// (optional) The metadata of the Stream Event.
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

	// GetStreamEvent returns a StreamEvent line by id.
	GetStreamEvent(tenantId string, streamEventId int64) (*dbsqlc.StreamEvent, error)

	// DeleteStreamEvent deletes a StreamEvent line by id.
	DeleteStreamEvent(tenantId string, streamEventId int64) (*dbsqlc.StreamEvent, error)

	// CleanupStreamEvents deletes all stale StreamEvents.
	CleanupStreamEvents() error
}
