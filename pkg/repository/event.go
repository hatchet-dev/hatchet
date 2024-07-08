package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type CreateEventOpts struct {
	// (required) the tenant id
	TenantId string `validate:"required,uuid"`

	// (required) the event key
	Key string `validate:"required"`

	// (optional) the event data
	Data []byte

	// (optional) the event that this event is replaying
	ReplayedEvent *string `validate:"omitempty,uuid"`

	// (optional) the event metadata
	AdditionalMetadata []byte
}

type ListEventOpts struct {
	// (optional) a list of event keys to filter by
	Keys []string

	// (optional) a list of workflow IDs to filter by
	Workflows []string

	// (optional) a list of workflow run statuses to filter by
	WorkflowRunStatus []db.WorkflowRunStatus

	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int

	// (optional) a search query
	Search *string

	// (optional) the event that this event is replaying
	ReplayedEvent *string `validate:"omitempty,uuid"`

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=createdAt"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`

	// (optional) the event metadata
	AdditionalMetadata []byte
}

type ListEventResult struct {
	Rows  []*dbsqlc.ListEventsRow
	Count int
}

type EventAPIRepository interface {
	// ListEvents returns all events for a given tenant.
	ListEvents(tenantId string, opts *ListEventOpts) (*ListEventResult, error)

	// ListEventKeys returns all unique event keys for a given tenant.
	ListEventKeys(tenantId string) ([]string, error)

	// GetEventById returns an event by id.
	GetEventById(id string) (*db.EventModel, error)

	// ListEventsById returns a list of events by id.
	ListEventsById(tenantId string, ids []string) ([]db.EventModel, error)
}

type EventEngineRepository interface {
	RegisterCreateCallback(callback Callback[*dbsqlc.Event])

	// CreateEvent creates a new event for a given tenant.
	CreateEvent(ctx context.Context, opts *CreateEventOpts) (*dbsqlc.Event, error)

	// GetEventForEngine returns an event for the engine by id.
	GetEventForEngine(ctx context.Context, tenantId, id string) (*dbsqlc.Event, error)

	ListEventsByIds(ctx context.Context, tenantId string, ids []string) ([]*dbsqlc.Event, error)

	// DeleteExpiredEvents deletes events that were created before the given time. It returns the number of deleted events
	// and the number of non-deleted events that match the conditions.
	DeleteExpiredEvents(ctx context.Context, tenantId string, before time.Time) (int, int, error)
}
