package ingestor

import (
	"context"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"

	msgqueuev1 "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type Ingestor interface {
	contracts.EventsServiceServer
	IngestEvent(ctx context.Context, tenant *dbsqlc.Tenant, eventName string, data []byte, metadata []byte, priority *int32, scope, triggeringWebhookName *string) (*dbsqlc.Event, error)
	IngestWebhookValidationFailure(ctx context.Context, tenant *dbsqlc.Tenant, webhookName, errorText string) error
	BulkIngestEvent(ctx context.Context, tenant *dbsqlc.Tenant, eventOpts []*CreateEventOpts) ([]*dbsqlc.Event, error)
	IngestReplayedEvent(ctx context.Context, tenant *dbsqlc.Tenant, replayedEvent *dbsqlc.Event) (*dbsqlc.Event, error)
	IngestCELEvaluationFailure(ctx context.Context, tenantId, errorText string, source sqlcv1.V1CelEvaluationFailureSource) error
}

type IngestorOptFunc func(*IngestorOpts)

type IngestorOpts struct {
	eventRepository        repository.EventEngineRepository
	entitlementsRepository repository.EntitlementsRepository
	stepRunRepository      repository.StepRunEngineRepository
	mqv1                   msgqueuev1.MessageQueue
	repov1                 v1.Repository
	isLogIngestionEnabled  bool
}

func WithEventRepository(r repository.EventEngineRepository) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.eventRepository = r
	}
}

func WithEntitlementsRepository(r repository.EntitlementsRepository) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.entitlementsRepository = r
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

func WithLogIngestionEnabled(isEnabled bool) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.isLogIngestionEnabled = isEnabled
	}
}

func defaultIngestorOpts() *IngestorOpts {
	return &IngestorOpts{
		isLogIngestionEnabled: true,
	}
}

type IngestorImpl struct {
	contracts.UnimplementedEventsServiceServer

	logRepository            repository.LogsEngineRepository
	streamEventRepository    repository.StreamEventsEngineRepository
	entitlementsRepository   repository.EntitlementsRepository
	stepRunRepository        repository.StepRunEngineRepository
	steprunTenantLookupCache *lru.Cache[string, string]

	mqv1   msgqueuev1.MessageQueue
	v      validator.Validator
	repov1 v1.Repository

	isLogIngestionEnabled bool
}

func NewIngestor(fs ...IngestorOptFunc) (Ingestor, error) {
	opts := defaultIngestorOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.eventRepository == nil {
		return nil, fmt.Errorf("event repository is required. use WithEventRepository")
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
		entitlementsRepository:   opts.entitlementsRepository,
		stepRunRepository:        opts.stepRunRepository,
		steprunTenantLookupCache: stepRunCache,
		mqv1:                     opts.mqv1,
		v:                        validator.NewDefaultValidator(),
		repov1:                   opts.repov1,
		isLogIngestionEnabled:    opts.isLogIngestionEnabled,
	}, nil
}

func (i *IngestorImpl) IngestEvent(ctx context.Context, tenant *dbsqlc.Tenant, key string, data []byte, metadata []byte, priority *int32, scope, triggeringWebhookName *string) (*dbsqlc.Event, error) {
	return i.ingestEventV1(ctx, tenant, key, data, metadata, priority, scope, triggeringWebhookName)
}

func (i *IngestorImpl) IngestWebhookValidationFailure(ctx context.Context, tenant *dbsqlc.Tenant, webhookName, errorText string) error {
	return i.ingestWebhookValidationFailure(tenant.ID.String(), webhookName, errorText)
}

func (i *IngestorImpl) BulkIngestEvent(ctx context.Context, tenant *dbsqlc.Tenant, eventOpts []*CreateEventOpts) ([]*dbsqlc.Event, error) {
	return i.bulkIngestEventV1(ctx, tenant, eventOpts)
}

func (i *IngestorImpl) IngestReplayedEvent(ctx context.Context, tenant *dbsqlc.Tenant, replayedEvent *dbsqlc.Event) (*dbsqlc.Event, error) {
	return i.ingestReplayedEventV1(ctx, tenant, replayedEvent)
}

func (i *IngestorImpl) IngestCELEvaluationFailure(ctx context.Context, tenantId, errorText string, source sqlcv1.V1CelEvaluationFailureSource) error {
	return i.ingestCELEvaluationFailure(
		ctx,
		tenantId,
		errorText,
		source,
	)
}
