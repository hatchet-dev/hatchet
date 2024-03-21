package ingestor

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
)

type Ingestor interface {
	contracts.EventsServiceServer
	IngestEvent(ctx context.Context, tenantId, eventName string, data []byte) (*dbsqlc.Event, error)
	IngestReplayedEvent(ctx context.Context, tenantId string, replayedEvent *dbsqlc.Event) (*dbsqlc.Event, error)
}

type IngestorOptFunc func(*IngestorOpts)

type IngestorOpts struct {
	eventRepository repository.EventEngineRepository
	logRepository   repository.LogsEngineRepository
	mq              msgqueue.MessageQueue
}

func WithEventRepository(r repository.EventEngineRepository) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.eventRepository = r
	}
}

func WithLogRepository(r repository.LogsEngineRepository) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.logRepository = r
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.mq = mq
	}
}

func defaultIngestorOpts() *IngestorOpts {
	return &IngestorOpts{}
}

type IngestorImpl struct {
	contracts.UnimplementedEventsServiceServer

	eventRepository repository.EventEngineRepository
	logRepository   repository.LogsEngineRepository
	mq              msgqueue.MessageQueue
}

func NewIngestor(fs ...IngestorOptFunc) (Ingestor, error) {
	opts := defaultIngestorOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.eventRepository == nil {
		return nil, fmt.Errorf("event repository is required. use WithEventRepository")
	}

	if opts.logRepository == nil {
		return nil, fmt.Errorf("log repository is required. use WithLogRepository")
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	return &IngestorImpl{
		eventRepository: opts.eventRepository,
		logRepository:   opts.logRepository,
		mq:              opts.mq,
	}, nil
}

func (i *IngestorImpl) IngestEvent(ctx context.Context, tenantId, key string, data []byte) (*dbsqlc.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "ingest-event")
	defer span.End()

	event, err := i.eventRepository.CreateEvent(ctx, &repository.CreateEventOpts{
		TenantId: tenantId,
		Key:      key,
		Data:     data,
	})

	if err != nil {
		return nil, fmt.Errorf("could not create event: %w", err)
	}

	telemetry.WithAttributes(span, telemetry.AttributeKV{
		Key:   "event_id",
		Value: event.ID,
	})

	err = i.mq.AddMessage(context.Background(), msgqueue.EVENT_PROCESSING_QUEUE, eventToTask(event))

	if err != nil {
		return nil, fmt.Errorf("could not add event to task queue: %w", err)
	}

	return event, nil
}

func (i *IngestorImpl) IngestReplayedEvent(ctx context.Context, tenantId string, replayedEvent *dbsqlc.Event) (*dbsqlc.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "ingest-replayed-event")
	defer span.End()

	replayedId := sqlchelpers.UUIDToStr(replayedEvent.ID)

	event, err := i.eventRepository.CreateEvent(ctx, &repository.CreateEventOpts{
		TenantId:      tenantId,
		Key:           replayedEvent.Key,
		Data:          replayedEvent.Data,
		ReplayedEvent: &replayedId,
	})

	if err != nil {
		return nil, fmt.Errorf("could not create event: %w", err)
	}

	err = i.mq.AddMessage(context.Background(), msgqueue.EVENT_PROCESSING_QUEUE, eventToTask(event))

	if err != nil {
		return nil, fmt.Errorf("could not add event to task queue: %w", err)
	}

	return event, nil
}

func eventToTask(e *dbsqlc.Event) *msgqueue.Message {
	eventId := sqlchelpers.UUIDToStr(e.ID)
	tenantId := sqlchelpers.UUIDToStr(e.TenantId)

	payloadTyped := tasktypes.EventTaskPayload{
		EventId:   eventId,
		EventKey:  e.Key,
		EventData: string(e.Data),
	}

	payload, _ := datautils.ToJSONMap(payloadTyped)

	metadata, _ := datautils.ToJSONMap(tasktypes.EventTaskMetadata{
		EventKey: e.Key,
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "event",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}
