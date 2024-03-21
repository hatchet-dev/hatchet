package repository

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type CreateDispatcherOpts struct {
	ID string `validate:"required,uuid"`
}

type UpdateDispatcherOpts struct {
	LastHeartbeatAt *time.Time
}

type DispatcherEngineRepository interface {
	// CreateNewDispatcher creates a new dispatcher for a given tenant.
	CreateNewDispatcher(opts *CreateDispatcherOpts) (*dbsqlc.Dispatcher, error)

	// UpdateDispatcher updates a dispatcher for a given tenant.
	UpdateDispatcher(dispatcherId string, opts *UpdateDispatcherOpts) (*dbsqlc.Dispatcher, error)

	Delete(dispatcherId string) error

	UpdateStaleDispatchers(onStale func(dispatcherId string, getValidDispatcherId func() string) error) error
}
