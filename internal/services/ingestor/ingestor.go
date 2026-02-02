package ingestor

import (
	"context"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/task/trigger"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/scheduler/v1"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	"github.com/rs/zerolog"
)

type Ingestor interface {
	contracts.EventsServiceServer
	IngestEvent(ctx context.Context, tenant *sqlcv1.Tenant, eventName string, data []byte, metadata []byte, priority *int32, scope, triggeringWebhookName *string) (*sqlcv1.Event, error)
	IngestWebhookValidationFailure(ctx context.Context, tenant *sqlcv1.Tenant, webhookName, errorText string) error
	BulkIngestEvent(ctx context.Context, tenant *sqlcv1.Tenant, eventOpts []*CreateEventOpts) ([]*sqlcv1.Event, error)
	IngestReplayedEvent(ctx context.Context, tenant *sqlcv1.Tenant, replayedEvent *sqlcv1.Event) (*sqlcv1.Event, error)
	IngestCELEvaluationFailure(ctx context.Context, tenantId, errorText string, source sqlcv1.V1CelEvaluationFailureSource) error
	Cleanup() error
}

type IngestorOptFunc func(*IngestorOpts)

type IngestorOpts struct {
	mqv1                  msgqueue.MessageQueue
	repov1                v1.Repository
	isLogIngestionEnabled bool

	localScheduler              *scheduler.Scheduler
	localDispatcher             *dispatcher.DispatcherImpl
	optimisticSchedulingEnabled bool
	l                           *zerolog.Logger

	grpcTriggersEnabled bool
	grpcTriggerSlots    int
}

func WithMessageQueueV1(mq msgqueue.MessageQueue) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.mqv1 = mq
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

func WithGrpcTriggersEnabled(enabled bool) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.grpcTriggersEnabled = enabled
	}
}

func WithGrpcTriggerSlots(slots int) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.grpcTriggerSlots = slots
	}
}

func defaultIngestorOpts() *IngestorOpts {
	l := logger.NewDefaultLogger("ingestor")

	return &IngestorOpts{
		isLogIngestionEnabled: true,
		l:                     &l,
	}
}

func WithOptimisticSchedulingEnabled(enabled bool) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.optimisticSchedulingEnabled = enabled
	}
}

func WithLocalScheduler(s *scheduler.Scheduler) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.localScheduler = s
	}
}

func WithLocalDispatcher(d *dispatcher.DispatcherImpl) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.localDispatcher = d
	}
}

func WithLogger(l *zerolog.Logger) IngestorOptFunc {
	return func(opts *IngestorOpts) {
		opts.l = l
	}
}

type IngestorImpl struct {
	contracts.UnimplementedEventsServiceServer

	steprunTenantLookupCache *lru.Cache[string, string]

	mqv1   msgqueue.MessageQueue
	v      validator.Validator
	repov1 v1.Repository

	isLogIngestionEnabled bool

	localScheduler  *scheduler.Scheduler
	localDispatcher *dispatcher.DispatcherImpl
	l               *zerolog.Logger

	tw        *trigger.TriggerWriter
	pubBuffer *msgqueue.MQPubBuffer
}

func NewIngestor(fs ...IngestorOptFunc) (Ingestor, error) {
	opts := defaultIngestorOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mqv1 == nil {
		return nil, fmt.Errorf("task queue v1 is required. use WithMessageQueueV1")
	}

	if opts.repov1 == nil {
		return nil, fmt.Errorf("repository v1 is required. use WithRepositoryV1")
	}

	// estimate of 1000 * 2 * UUID string size (roughly 104kb max)
	stepRunCache, err := lru.New[string, string](1000)

	if err != nil {
		return nil, fmt.Errorf("could not create step run cache: %w", err)
	}

	var tw *trigger.TriggerWriter
	var pubBuffer *msgqueue.MQPubBuffer

	if opts.grpcTriggersEnabled {
		pubBuffer = msgqueue.NewMQPubBuffer(opts.mqv1)

		tw = trigger.NewTriggerWriter(opts.mqv1, opts.repov1, opts.l, pubBuffer, opts.grpcTriggerSlots)
	}

	var localScheduler *scheduler.Scheduler

	if opts.optimisticSchedulingEnabled && opts.localScheduler != nil {
		localScheduler = opts.localScheduler
	} else if opts.optimisticSchedulingEnabled && opts.localScheduler == nil {
		return nil, fmt.Errorf("optimistic writes enabled but no local scheduler provided")
	}

	return &IngestorImpl{
		steprunTenantLookupCache: stepRunCache,
		mqv1:                     opts.mqv1,
		v:                        validator.NewDefaultValidator(),
		repov1:                   opts.repov1,
		isLogIngestionEnabled:    opts.isLogIngestionEnabled,
		l:                        opts.l,
		localScheduler:           localScheduler,
		localDispatcher:          opts.localDispatcher,
		tw:                       tw,
		pubBuffer:                pubBuffer,
	}, nil
}

func (i *IngestorImpl) IngestEvent(ctx context.Context, tenant *sqlcv1.Tenant, key string, data []byte, metadata []byte, priority *int32, scope, triggeringWebhookName *string) (*sqlcv1.Event, error) {
	return i.ingestEventV1(ctx, tenant, key, data, metadata, priority, scope, triggeringWebhookName)
}

func (i *IngestorImpl) IngestWebhookValidationFailure(ctx context.Context, tenant *sqlcv1.Tenant, webhookName, errorText string) error {
	return i.ingestWebhookValidationFailure(tenant.ID.String(), webhookName, errorText)
}

func (i *IngestorImpl) BulkIngestEvent(ctx context.Context, tenant *sqlcv1.Tenant, eventOpts []*CreateEventOpts) ([]*sqlcv1.Event, error) {
	return i.bulkIngestEventV1(ctx, tenant, eventOpts)
}

func (i *IngestorImpl) IngestReplayedEvent(ctx context.Context, tenant *sqlcv1.Tenant, replayedEvent *sqlcv1.Event) (*sqlcv1.Event, error) {
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

// Cleanup stops the pubBuffer goroutines if they exist
func (i *IngestorImpl) Cleanup() error {
	if i.pubBuffer != nil {
		i.pubBuffer.Stop()
	}
	return nil
}
