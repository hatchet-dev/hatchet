package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateDispatcherOpts struct {
	ID uuid.UUID `validate:"required"`
}

type DispatcherRepository interface {
	// CreateNewDispatcher creates a new dispatcher for a given tenant.
	CreateNewDispatcher(ctx context.Context, opts *CreateDispatcherOpts) (*sqlcv1.Dispatcher, error)

	// Heartbeat records that the dispatcher is alive and can write to the database. A heartbeat
	// that follows a gap of more than 15 seconds restarts the dispatcher's unbroken run, which
	// pauses task reassignment (see TaskRepository.ProcessTaskReassignments).
	Heartbeat(ctx context.Context, dispatcherId uuid.UUID) error

	Delete(ctx context.Context, dispatcherId uuid.UUID) error

	// MarkInactive flags the dispatcher as shutting down. ClaimOperators then claims nothing
	// for it and lets other dispatchers take its operators over at once.
	MarkInactive(ctx context.Context, dispatcherId uuid.UUID) error
}

type dispatcherRepository struct {
	*sharedRepository
}

func newDispatcherRepository(shared *sharedRepository) DispatcherRepository {
	return &dispatcherRepository{
		sharedRepository: shared,
	}
}

func (d *dispatcherRepository) CreateNewDispatcher(ctx context.Context, opts *CreateDispatcherOpts) (*sqlcv1.Dispatcher, error) {
	if err := d.v.Validate(opts); err != nil {
		return nil, err
	}

	return d.queries.CreateDispatcher(ctx, d.pool.ForShared(), opts.ID)
}

func (d *dispatcherRepository) Heartbeat(ctx context.Context, dispatcherId uuid.UUID) error {
	return d.queries.HeartbeatDispatcher(ctx, d.pool.ForShared(), dispatcherId)
}

func (d *dispatcherRepository) Delete(ctx context.Context, dispatcherId uuid.UUID) error {
	_, err := d.queries.DeleteDispatcher(ctx, d.pool.ForShared(), dispatcherId)
	return err
}

func (d *dispatcherRepository) MarkInactive(ctx context.Context, dispatcherId uuid.UUID) error {
	_, err := d.queries.SetDispatchersInactive(ctx, d.pool.ForShared(), []uuid.UUID{dispatcherId})
	return err
}
