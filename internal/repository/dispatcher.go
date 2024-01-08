package repository

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

type CreateDispatcherOpts struct {
	ID string `validate:"required,uuid"`
}

type UpdateDispatcherOpts struct {
	LastHeartbeatAt *time.Time
}

type DispatcherRepository interface {
	// GetDispatcherForWorker returns the dispatcher connected to a given worker.
	GetDispatcherForWorker(workerId string) (*db.DispatcherModel, error)

	// CreateNewDispatcher creates a new dispatcher for a given tenant.
	CreateNewDispatcher(opts *CreateDispatcherOpts) (*db.DispatcherModel, error)

	// UpdateDispatcher updates a dispatcher for a given tenant.
	UpdateDispatcher(dispatcherId string, opts *UpdateDispatcherOpts) (*db.DispatcherModel, error)

	Delete(dispatcherId string) error

	// AddWorker adds a worker to a dispatcher.
	AddWorker(dispatcherId, workerId string) (*db.DispatcherModel, error)

	UpdateStaleDispatchers(onStale func(dispatcherId string, getValidDispatcherId func() string) error) error
}
