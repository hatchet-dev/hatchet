package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type CreateStreamEventOpts struct {
	// The step run id
	StepRunId string `validate:"required,uuid"`

	// (optional) The time when the StreamEvent was created.
	CreatedAt *time.Time

	// (required) The message of the Stream Event.
	Message []byte

	// (optional) The metadata of the Stream Event.
	Metadata []byte
}

type StreamEventsEngineRepository interface {
	// PutStreamEvent creates a new StreamEvent line.
	PutStreamEvent(ctx context.Context, tenantId string, opts *CreateStreamEventOpts) (*dbsqlc.StreamEvent, error)

	// GetStreamEvent returns a StreamEvent line by id.
	GetStreamEvent(ctx context.Context, tenantId string, streamEventId int64) (*dbsqlc.StreamEvent, error)

	// CleanupStreamEvents deletes all stale StreamEvents.
	CleanupStreamEvents(ctx context.Context) error

	// GetStreamEventMeta
	GetStreamEventMeta(ctx context.Context, tenantId string, stepRunId string) (*dbsqlc.GetStreamEventMetaRow, error)
}
