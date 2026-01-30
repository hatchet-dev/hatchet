package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateDispatcherOpts struct {
	ID string `validate:"required,uuid"`
}

type UpdateDispatcherOpts struct {
	LastHeartbeatAt *time.Time
}

type DispatcherRepository interface {
	// CreateNewDispatcher creates a new dispatcher for a given tenant.
	CreateNewDispatcher(ctx context.Context, opts *CreateDispatcherOpts) (*sqlcv1.Dispatcher, error)

	// UpdateDispatcher updates a dispatcher for a given tenant.
	UpdateDispatcher(ctx context.Context, dispatcherId string, opts *UpdateDispatcherOpts) (*sqlcv1.Dispatcher, error)

	Delete(ctx context.Context, dispatcherId string) error

	UpdateStaleDispatchers(ctx context.Context, onStale func(dispatcherId string, getValidDispatcherId func() string) error) error
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

	return d.queries.CreateDispatcher(ctx, d.pool, uuid.MustParse(opts.ID))
}

func (d *dispatcherRepository) UpdateDispatcher(ctx context.Context, dispatcherId string, opts *UpdateDispatcherOpts) (*sqlcv1.Dispatcher, error) {
	if err := d.v.Validate(opts); err != nil {
		return nil, err
	}

	return d.queries.UpdateDispatcher(ctx, d.pool, sqlcv1.UpdateDispatcherParams{
		ID:              uuid.MustParse(dispatcherId),
		LastHeartbeatAt: sqlchelpers.TimestampFromTime(opts.LastHeartbeatAt.UTC()),
	})
}

func (d *dispatcherRepository) Delete(ctx context.Context, dispatcherId string) error {
	_, err := d.queries.DeleteDispatcher(ctx, d.pool, uuid.MustParse(dispatcherId))

	return err
}

func (d *dispatcherRepository) UpdateStaleDispatchers(ctx context.Context, onStale func(dispatcherId string, getValidDispatcherId func() string) error) error {
	tx, err := d.pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer sqlchelpers.DeferRollback(context.Background(), d.l, tx.Rollback)

	staleDispatchers, err := d.queries.ListStaleDispatchers(context.Background(), tx)

	if err != nil {
		return err
	}

	activeDispatchers, err := d.queries.ListActiveDispatchers(context.Background(), tx)

	if err != nil {
		return err
	}

	dispatchersToDelete := make([]uuid.UUID, 0)

	for i, dispatcher := range staleDispatchers {
		err := onStale(sqlchelpers.UUIDToStr(dispatcher.Dispatcher.ID), func() string {
			// assign tickers in round-robin fashion
			return sqlchelpers.UUIDToStr(activeDispatchers[i%len(activeDispatchers)].Dispatcher.ID)
		})

		if err != nil {
			return err
		}

		dispatchersToDelete = append(dispatchersToDelete, dispatcher.Dispatcher.ID)
	}

	_, err = d.queries.SetDispatchersInactive(context.Background(), tx, dispatchersToDelete)

	if err != nil {
		return err
	}

	return tx.Commit(context.Background())
}
