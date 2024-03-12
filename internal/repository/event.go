package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type CreateEventOpts struct {
	// (required) the tenant id
	TenantId string `validate:"required,uuid"`

	// (required) the event key
	Key string `validate:"required"`

	// (optional) the event data
	Data *db.JSON

	// (optional) the event that this event is replaying
	ReplayedEvent *string `validate:"omitempty,uuid"`
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
}

type ListEventResult struct {
	Rows  []*dbsqlc.ListEventsRow
	Count int
}

type EventRepository interface {
	// ListEvents returns all events for a given tenant.
	ListEvents(tenantId string, opts *ListEventOpts) (*ListEventResult, error)

	// ListEventKeys returns all unique event keys for a given tenant.
	ListEventKeys(tenantId string) ([]string, error)

	// GetEventById returns an event by id.
	GetEventById(id string) (*db.EventModel, error)

	// GetEventForEngine returns an event for the engine by id.
	GetEventForEngine(tenantId, id string) (*dbsqlc.GetEventForEngineRow, error)

	// ListEventsById returns a list of events by id.
	ListEventsById(tenantId string, ids []string) ([]db.EventModel, error)

	// CreateEvent creates a new event for a given tenant.
	CreateEvent(ctx context.Context, opts *CreateEventOpts) (*db.EventModel, error)
}
