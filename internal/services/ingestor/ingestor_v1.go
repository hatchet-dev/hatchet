package ingestor

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type EventResult struct {
	TenantId           string
	EventId            string
	EventKey           string
	Data               string
	AdditionalMetadata string
}

func (i *IngestorImpl) ingestEventV1(ctx context.Context, tenant *dbsqlc.Tenant, key string, data []byte, metadata []byte) (*dbsqlc.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "ingest-event")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	return i.ingestSingleton(tenantId, key, data, metadata)
}

func (i *IngestorImpl) ingestSingleton(tenantId, key string, data []byte, metadata []byte) (*dbsqlc.Event, error) {
	eventId := uuid.New().String()

	msg, err := eventToTaskV1(
		tenantId,
		eventId,
		key,
		data,
		metadata,
	)

	if err != nil {
		return nil, fmt.Errorf("could not create event task: %w", err)
	}

	err = i.mqv1.SendMessage(context.Background(), msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return nil, fmt.Errorf("could not add event to task queue: %w", err)
	}

	return &dbsqlc.Event{
		ID:                 sqlchelpers.UUIDFromStr(eventId),
		TenantId:           sqlchelpers.UUIDFromStr(tenantId),
		Key:                key,
		Data:               data,
		AdditionalMetadata: metadata,
	}, nil
}

func (i *IngestorImpl) bulkIngestEventV1(ctx context.Context, tenant *dbsqlc.Tenant, eventOpts []*repository.CreateEventOpts) ([]*dbsqlc.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "bulk-ingest-event")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	results := make([]*dbsqlc.Event, 0, len(eventOpts))

	for _, event := range eventOpts {
		res, err := i.ingestSingleton(tenantId, event.Key, event.Data, event.AdditionalMetadata)

		if err != nil {
			return nil, fmt.Errorf("could not ingest event: %w", err)
		}

		results = append(results, res)
	}

	return results, nil
}

func (i *IngestorImpl) ingestReplayedEventV1(ctx context.Context, tenant *dbsqlc.Tenant, replayedEvent *dbsqlc.Event) (*dbsqlc.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "ingest-replayed-event")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	return i.ingestSingleton(tenantId, replayedEvent.Key, replayedEvent.Data, replayedEvent.AdditionalMetadata)
}

func eventToTaskV1(tenantId, eventId, key string, data, additionalMeta []byte) (*msgqueue.Message, error) {
	payloadTyped := tasktypes.UserEventTaskPayload{
		EventId:                 eventId,
		EventKey:                key,
		EventData:               data,
		EventAdditionalMetadata: additionalMeta,
	}

	return msgqueue.NewTenantMessage(
		tenantId,
		"user-event",
		false,
		true,
		payloadTyped,
	)
}
