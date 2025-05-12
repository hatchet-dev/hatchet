package ingestor

import (
	"context"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	msgqueuev1 "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type Ingestor interface {
	contracts.EventsServiceServer
	IngestEvent(ctx context.Context, tenant *dbsqlc.Tenant, eventName string, data []byte, metadata []byte) (*dbsqlc.Event, error)
	BulkIngestEvent(ctx context.Context, tenant *dbsqlc.Tenant, eventOpts []*repository.CreateEventOpts) ([]*dbsqlc.Event, error)
	IngestReplayedEvent(ctx context.Context, tenant *dbsqlc.Tenant, replayedEvent *dbsqlc.Event) (*dbsqlc.Event, error)
}

type IngestorOptFunc func(*IngestorOpts)

type IngestorOpts struct {
	eventRepository        repository.EventEngineRepository
	streamEventRepository  repository.StreamEventsEngineRepository
	logRepository          repository.LogsEngineRepository
	entitlementsRepository repository.EntitlementsRepository
	stepRunRepository      repository.StepRunEngineRepository
	mq                     msgqueue.MessageQueue
	mqv1                   msgqueuev1.MessageQueue
	repov1                 v1.Repository
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

func WithMessageQueueV1(mq msgqueuev1.MessageQueue) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.mqv1 = mq
	}
}

func WithStepRunRepository(r repository.StepRunEngineRepository) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.stepRunRepository = r
	}
}

func WithRepositoryV1(r v1.Repository) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.repov1 = r
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

	mq     msgqueue.MessageQueue
	mqv1   msgqueuev1.MessageQueue
	v      validator.Validator
	repov1 v1.Repository
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

	if opts.mqv1 == nil {
		return nil, fmt.Errorf("task queue v1 is required. use WithMessageQueueV1")
	}

	if opts.repov1 == nil {
		return nil, fmt.Errorf("repository v1 is required. use WithRepositoryV1")
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
		logRepository:            opts.logRepository,
		mq:                       opts.mq,
		mqv1:                     opts.mqv1,
		v:                        validator.NewDefaultValidator(),
		repov1:                   opts.repov1,
	}, nil
}

func (i *IngestorImpl) IngestEvent(ctx context.Context, tenant *dbsqlc.Tenant, key string, data []byte, metadata []byte) (*dbsqlc.Event, error) {
	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return i.ingestEventV0(ctx, tenant, key, data, metadata)
	case dbsqlc.TenantMajorEngineVersionV1:
		return i.ingestEventV1(ctx, tenant, key, data, metadata)
	default:
		return nil, fmt.Errorf("unsupported tenant version: %s", tenant.Version)
	}
}

func (i *IngestorImpl) ingestEventV0(ctx context.Context, tenant *dbsqlc.Tenant, key string, data []byte, metadata []byte) (*dbsqlc.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "ingest-event")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	event, err := i.eventRepository.CreateEvent(ctx, &repository.CreateEventOpts{
		TenantId:           tenantId,
		Key:                key,
		Data:               data,
		AdditionalMetadata: metadata,
	})

	if err == metered.ErrResourceExhausted {
		return nil, metered.ErrResourceExhausted
	}

	if err != nil {
		return nil, fmt.Errorf("could not create events: %w", err)
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

func (i *IngestorImpl) BulkIngestEvent(ctx context.Context, tenant *dbsqlc.Tenant, eventOpts []*repository.CreateEventOpts) ([]*dbsqlc.Event, error) {

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return i.bulkIngestEventV0(ctx, tenant, eventOpts)
	case dbsqlc.TenantMajorEngineVersionV1:
		return i.bulkIngestEventV1(ctx, tenant, eventOpts)
	default:
		return nil, fmt.Errorf("unsupported tenant version: %s", tenant.Version)
	}
}

func (i *IngestorImpl) bulkIngestEventV0(ctx context.Context, tenant *dbsqlc.Tenant, eventOpts []*repository.CreateEventOpts) ([]*dbsqlc.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "bulk-ingest-event")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	events, err := i.eventRepository.BulkCreateEvent(ctx, &repository.BulkCreateEventOpts{
		Events:   eventOpts,
		TenantId: tenantId,
	})

	if err == metered.ErrResourceExhausted {
		return nil, metered.ErrResourceExhausted
	}

	if err != nil {
		return nil, fmt.Errorf("could not create events: %w", err)
	}

	// TODO any attributes we want to add here? could jam in all the event ids? but could be a lot

	// telemetry.WithAttributes(span, telemetry.AttributeKV{
	// 	Key:   "event_id",
	// 	Value: event.ID,
	// })

	for _, event := range events.Events {
		err = i.mq.AddMessage(context.Background(), msgqueue.EVENT_PROCESSING_QUEUE, eventToTask(event))
		if err != nil {
			return nil, fmt.Errorf("could not add event to task queue: %w", err)
		}
	}

	return events.Events, nil
}

func (i *IngestorImpl) IngestReplayedEvent(ctx context.Context, tenant *dbsqlc.Tenant, replayedEvent *dbsqlc.Event) (*dbsqlc.Event, error) {

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return i.ingestReplayedEventV0(ctx, tenant, replayedEvent)
	case dbsqlc.TenantMajorEngineVersionV1:
		return i.ingestReplayedEventV1(ctx, tenant, replayedEvent)
	default:
		return nil, fmt.Errorf("unsupported tenant version: %s", tenant.Version)
	}
}

func (i *IngestorImpl) ingestReplayedEventV0(ctx context.Context, tenant *dbsqlc.Tenant, replayedEvent *dbsqlc.Event) (*dbsqlc.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "ingest-replayed-event")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	replayedId := sqlchelpers.UUIDToStr(replayedEvent.ID)

	event, err := i.eventRepository.CreateEvent(ctx, &repository.CreateEventOpts{
		TenantId:           tenantId,
		Key:                replayedEvent.Key,
		Data:               replayedEvent.Data,
		AdditionalMetadata: replayedEvent.AdditionalMetadata,
		ReplayedEvent:      &replayedId,
	})

	if err == metered.ErrResourceExhausted {
		return nil, metered.ErrResourceExhausted
	}

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
		EventId:                 eventId,
		EventKey:                e.Key,
		EventData:               string(e.Data),
		EventAdditionalMetadata: string(e.AdditionalMetadata),
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
