package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type EventWithPayload struct {
	*sqlcv1.V1Event
	Payload []byte
}

type EventsRepository interface {
	Get(ctx context.Context, externalId uuid.UUID) (*EventWithPayload, error)
}

type eventsRepository struct {
	*sharedRepository
}

func newEventsRepository(shared *sharedRepository) EventsRepository {
	return &eventsRepository{
		sharedRepository: shared,
	}
}

func (e *eventsRepository) Get(ctx context.Context, externalId uuid.UUID) (*EventWithPayload, error) {
	event, err := e.queries.GetEvent(ctx, e.pool, externalId)

	if err != nil {
		return nil, err
	}

	payload, err := e.payloadStore.RetrieveSingle(ctx, nil, RetrievePayloadOpts{
		Id:         event.ID,
		InsertedAt: event.SeenAt,
		Type:       sqlcv1.V1PayloadTypeUSEREVENTINPUT,
		TenantId:   event.TenantID,
	})

	if err != nil {
		return nil, err
	}

	return &EventWithPayload{
		V1Event: event,
		Payload: payload,
	}, nil
}
