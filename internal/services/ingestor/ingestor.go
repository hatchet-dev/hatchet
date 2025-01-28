package ingestor

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type Ingestor interface {
	contracts.EventsServiceServer
	IngestEvent(ctx context.Context, tenantId, eventName string, data []byte, metadata []byte) (*EventResult, error)
	BulkIngestEvent(ctx context.Context, tenantID string, eventOpts []*repository.CreateEventOpts) ([]*EventResult, error)
	IngestReplayedEvent(ctx context.Context, tenantId string, replayedEvent *dbsqlc.Event) (*EventResult, error)
}

type IngestorOptFunc func(*IngestorOpts)

type IngestorOpts struct {
	eventRepository        repository.EventEngineRepository
	streamEventRepository  repository.StreamEventsEngineRepository
	logRepository          repository.LogsEngineRepository
	entitlementsRepository repository.EntitlementsRepository
	stepRunRepository      repository.StepRunEngineRepository
	mq                     msgqueue.MessageQueue
}

func WithEventRepository(r repository.EventEngineRepository) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.eventRepository = r
	}
}

func WithStreamEventsRepository(r repository.StreamEventsEngineRepository) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.streamEventRepository = r
	}
}

func WithLogRepository(r repository.LogsEngineRepository) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.logRepository = r
	}
}

func WithEntitlementsRepository(r repository.EntitlementsRepository) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.entitlementsRepository = r
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.mq = mq
	}
}

func WithStepRunRepository(r repository.StepRunEngineRepository) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.stepRunRepository = r
	}
}

func defaultIngestorOpts() *IngestorOpts {
	return &IngestorOpts{}
}

type IngestorImpl struct {
	contracts.UnimplementedEventsServiceServer

	eventRepository          repository.EventEngineRepository
	logRepository            repository.LogsEngineRepository
	streamEventRepository    repository.StreamEventsEngineRepository
	entitlementsRepository   repository.EntitlementsRepository
	stepRunRepository        repository.StepRunEngineRepository
	steprunTenantLookupCache *lru.Cache[string, string]

	mq msgqueue.MessageQueue
	v  validator.Validator
}

func NewIngestor(fs ...IngestorOptFunc) (Ingestor, error) {
	opts := defaultIngestorOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.eventRepository == nil {
		return nil, fmt.Errorf("event repository is required. use WithEventRepository")
	}

	if opts.streamEventRepository == nil {
		return nil, fmt.Errorf("stream event repository is required. use WithStreamEventRepository")
	}

	if opts.logRepository == nil {
		return nil, fmt.Errorf("log repository is required. use WithLogRepository")
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.stepRunRepository == nil {
		return nil, fmt.Errorf("step run repository is required. use WithStepRunRepository")
	}
	// estimate of 1000 * 2 * UUID string size (roughly 104kb max)
	stepRunCache, err := lru.New[string, string](1000)

	if err != nil {
		return nil, fmt.Errorf("could not create step run cache: %w", err)
	}

	return &IngestorImpl{
		eventRepository:          opts.eventRepository,
		streamEventRepository:    opts.streamEventRepository,
		entitlementsRepository:   opts.entitlementsRepository,
		stepRunRepository:        opts.stepRunRepository,
		steprunTenantLookupCache: stepRunCache,

		logRepository: opts.logRepository,
		mq:            opts.mq,
		v:             validator.NewDefaultValidator(),
	}, nil
}

type EventResult struct {
	TenantId           string
	EventId            string
	EventKey           string
	Data               string
	AdditionalMetadata string
}

func (i *IngestorImpl) IngestEvent(ctx context.Context, tenantId, key string, data []byte, metadata []byte) (*EventResult, error) {
	ctx, span := telemetry.NewSpan(ctx, "ingest-event")
	defer span.End()

	return i.ingestSingleton(tenantId, key, data, metadata)
}

func (i *IngestorImpl) ingestSingleton(tenantId, key string, data []byte, metadata []byte) (*EventResult, error) {
	eventId := uuid.New().String()

	msg, err := eventToTask(
		tenantId,
		eventId,
		key,
		string(data),
		string(metadata),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create event task: %w", err)
	}

	err = i.mq.SendMessage(context.Background(), msgqueue.TRIGGER_QUEUE, msg)

	if err != nil {
		return nil, fmt.Errorf("could not add event to task queue: %w", err)
	}

	return &EventResult{
		TenantId:           tenantId,
		EventId:            eventId,
		EventKey:           key,
		Data:               string(data),
		AdditionalMetadata: string(metadata),
	}, nil
}

func (i *IngestorImpl) BulkIngestEvent(ctx context.Context, tenantId string, eventOpts []*repository.CreateEventOpts) ([]*EventResult, error) {
	ctx, span := telemetry.NewSpan(ctx, "bulk-ingest-event")
	defer span.End()

	// events, err := i.eventRepository.BulkCreateEvent(ctx, &repository.BulkCreateEventOpts{
	// 	Events:   eventOpts,
	// 	TenantId: tenantId,
	// })

	// if err == metered.ErrResourceExhausted {
	// 	return nil, metered.ErrResourceExhausted
	// }

	// if err != nil {
	// 	return nil, fmt.Errorf("could not create events: %w", err)
	// }

	// TODO any attributes we want to add here? could jam in all the event ids? but could be a lot

	// telemetry.WithAttributes(span, telemetry.AttributeKV{
	// 	Key:   "event_id",
	// 	Value: event.ID,
	// })

	results := make([]*EventResult, 0, len(eventOpts))

	for _, event := range eventOpts {
		res, err := i.ingestSingleton(tenantId, event.Key, event.Data, event.AdditionalMetadata)

		if err != nil {
			return nil, fmt.Errorf("could not ingest event: %w", err)
		}

		results = append(results, res)
	}

	return results, nil
}

func (i *IngestorImpl) IngestReplayedEvent(ctx context.Context, tenantId string, replayedEvent *dbsqlc.Event) (*EventResult, error) {
	ctx, span := telemetry.NewSpan(ctx, "ingest-replayed-event")
	defer span.End()

	return i.ingestSingleton(tenantId, replayedEvent.Key, replayedEvent.Data, replayedEvent.AdditionalMetadata)
}

func eventToTask(tenantId, eventId, key, data, additionalMeta string) (*msgqueue.Message, error) {
	payloadTyped := tasktypes.EventTaskPayload{
		EventId:                 eventId,
		EventKey:                key,
		EventData:               string(data),
		EventAdditionalMetadata: string(additionalMeta),
	}

	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"event-trigger",
		payloadTyped,
		false,
	)
}
