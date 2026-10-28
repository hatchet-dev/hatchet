package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateDispatcherOpts struct {
	ID uuid.UUID `validate:"required"`

	// BrokerID identifies the message broker the dispatcher consumes its queue from; "" when unknown.
	BrokerID string
}

type UpdateDispatcherOpts struct {
	LastHeartbeatAt *time.Time

	// BrokerID replaces the stored broker identity unless it is "".
	BrokerID string
}

type DispatcherRepository interface {
	// CreateNewDispatcher creates a new dispatcher for a given tenant.
	CreateNewDispatcher(ctx context.Context, opts *CreateDispatcherOpts) (*sqlcv1.Dispatcher, error)

	// UpdateDispatcher updates a dispatcher for a given tenant.
	UpdateDispatcher(ctx context.Context, dispatcherId uuid.UUID, opts *UpdateDispatcherOpts) (*sqlcv1.Dispatcher, error)

	Delete(ctx context.Context, dispatcherId uuid.UUID) error
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

	return d.queries.CreateDispatcher(ctx, d.pool.ForShared(), sqlcv1.CreateDispatcherParams{
		ID:       opts.ID,
		BrokerId: brokerIDText(opts.BrokerID),
	})
}

func (d *dispatcherRepository) UpdateDispatcher(ctx context.Context, dispatcherId uuid.UUID, opts *UpdateDispatcherOpts) (*sqlcv1.Dispatcher, error) {
	if err := d.v.Validate(opts); err != nil {
		return nil, err
	}

	return d.queries.UpdateDispatcher(ctx, d.pool.ForShared(), sqlcv1.UpdateDispatcherParams{
		ID:              dispatcherId,
		LastHeartbeatAt: sqlchelpers.TimestampFromTime(opts.LastHeartbeatAt.UTC()),
		BrokerId:        brokerIDText(opts.BrokerID),
	})
}

// brokerIDText maps an unknown ("") broker identity to NULL.
func brokerIDText(brokerID string) pgtype.Text {
	return pgtype.Text{String: brokerID, Valid: brokerID != ""}
}

func (d *dispatcherRepository) Delete(ctx context.Context, dispatcherId uuid.UUID) error {
	_, err := d.queries.DeleteDispatcher(ctx, d.pool.ForShared(), dispatcherId)
	return err
}
