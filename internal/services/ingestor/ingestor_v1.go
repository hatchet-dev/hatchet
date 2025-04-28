package ingestor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
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

	canCreateEvents, eLimit, err := i.entitlementsRepository.TenantLimit().CanCreate(
		ctx,
		dbsqlc.LimitResourceEVENT,
		tenantId,
		1,
	)

	if err != nil {
		return nil, fmt.Errorf("could not check tenant limit: %w", err)
	}

	if !canCreateEvents {
		return nil, status.Error(
			codes.ResourceExhausted,
			fmt.Sprintf("tenant has reached %d%% of its events limit", eLimit),
		)
	}

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

	now := time.Now().UTC()

	olapEvent, err := i.repov1.OLAP().CreateEvent(
		context.Background(),
		tenantId,
		sqlcv1.CreateEventParams{
			Tenantid:           sqlchelpers.UUIDFromStr(tenantId),
			Generatedat:        sqlchelpers.TimestamptzFromTime(now),
			Key:                key,
			Payload:            data,
			AdditionalMetadata: metadata,
		},
	)

	if err != nil || olapEvent == nil {
		return nil, fmt.Errorf("could not create event in OLAP: %w", err)
	}

	return &dbsqlc.Event{
		ID:                 sqlchelpers.UUIDFromStr(eventId),
		CreatedAt:          sqlchelpers.TimestampFromTime(now),
		UpdatedAt:          sqlchelpers.TimestampFromTime(now),
		Key:                key,
		TenantId:           sqlchelpers.UUIDFromStr(tenantId),
		Data:               data,
		AdditionalMetadata: metadata,
	}, nil
}

func (i *IngestorImpl) bulkIngestEventV1(ctx context.Context, tenant *dbsqlc.Tenant, eventOpts []*repository.CreateEventOpts) ([]*dbsqlc.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "bulk-ingest-event")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	count := len(eventOpts)

	canCreateEvents, eLimit, err := i.entitlementsRepository.TenantLimit().CanCreate(
		ctx,
		dbsqlc.LimitResourceEVENT,
		tenantId,
		int32(count), // nolint: gosec
	)

	if err != nil {
		return nil, fmt.Errorf("could not check tenant limit: %w", err)
	}

	if !canCreateEvents {
		return nil, status.Error(
			codes.ResourceExhausted,
			fmt.Sprintf("tenant has reached %d%% of its events limit", eLimit),
		)
	}

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
