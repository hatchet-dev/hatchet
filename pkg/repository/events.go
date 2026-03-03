package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type EventsRepository interface {
	Get(ctx context.Context, externalId uuid.UUID) (*sqlcv1.V1Event, error)
}

type eventsRepository struct {
	*sharedRepository
}

func newEventsRepository(shared *sharedRepository) EventsRepository {
	return &eventsRepository{
		sharedRepository: shared,
	}
}

func (e *eventsRepository) Get(ctx context.Context, externalId uuid.UUID) (*sqlcv1.V1Event, error) {
	return e.queries.GetEvent(ctx, e.pool, externalId)
}
