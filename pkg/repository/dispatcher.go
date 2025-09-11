package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type CreateDispatcherOpts struct {
	ID string `validate:"required,uuid"`
}

type UpdateDispatcherOpts struct {
	LastHeartbeatAt *time.Time
}

type DispatcherEngineRepository interface {
	// CreateNewDispatcher creates a new dispatcher for a given tenant.
	CreateNewDispatcher(ctx context.Context, opts *CreateDispatcherOpts) (*dbsqlc.Dispatcher, error)

	// UpdateDispatcher updates a dispatcher for a given tenant.
	UpdateDispatcher(ctx context.Context, dispatcherId string, opts *UpdateDispatcherOpts) (*dbsqlc.Dispatcher, error)

	Delete(ctx context.Context, dispatcherId string) error

	UpdateStaleDispatchers(ctx context.Context, onStale func(dispatcherId string, getValidDispatcherId func() string) error) error
}
