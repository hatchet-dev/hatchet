package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type EventsController interface {
	Start(ctx context.Context) error
}

type EventsControllerImpl struct {
	mq msgqueue.MessageQueue
	l  *zerolog.Logger

	entitlements repository.EntitlementsRepository

	repo repository.EngineRepository
	dv   datautils.DataDecoderValidator
}

type EventsControllerOpt func(*EventsControllerOpts)

type EventsControllerOpts struct {
	mq           msgqueue.MessageQueue
	l            *zerolog.Logger
	entitlements repository.EntitlementsRepository
	repo         repository.EngineRepository
	dv           datautils.DataDecoderValidator
}

func defaultEventsControllerOpts() *EventsControllerOpts {
	logger := logger.NewDefaultLogger("events-controller")
	return &EventsControllerOpts{
		l:  &logger,
		dv: datautils.NewDataDecoderValidator(),
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) EventsControllerOpt {
	return func(opts *EventsControllerOpts) {
		opts.mq = mq
	}
}

func WithLogger(l *zerolog.Logger) EventsControllerOpt {
	return func(opts *EventsControllerOpts) {
		opts.l = l
	}
}

func WithRepository(r repository.EngineRepository) EventsControllerOpt {
	return func(opts *EventsControllerOpts) {
		opts.repo = r
	}
}

func WithEntitlementsRepository(r repository.EntitlementsRepository) EventsControllerOpt {
	return func(opts *EventsControllerOpts) {
		opts.entitlements = r
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) EventsControllerOpt {
	return func(opts *EventsControllerOpts) {
		opts.dv = dv
	}
}

func New(fs ...EventsControllerOpt) (*EventsControllerImpl, error) {
	opts := defaultEventsControllerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.entitlements == nil {
		return nil, fmt.Errorf("entitlements repository is required. use WithEntitlementsRepository")
	}

	newLogger := opts.l.With().Str("service", "events-controller").Logger()
	opts.l = &newLogger

	return &EventsControllerImpl{
		mq:           opts.mq,
		l:            opts.l,
		repo:         opts.repo,
		entitlements: opts.entitlements,
		dv:           opts.dv,
	}, nil
}

func (ec *EventsControllerImpl) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	f := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		err := ec.handleTask(ctx, task)
		if err != nil {
			ec.l.Error().Err(err).Msgf("could not handle event task %s", task.ID)
			return err
		}

		return nil
	}

	cleanupQueue, err := ec.mq.Subscribe(msgqueue.EVENT_PROCESSING_QUEUE, f, msgqueue.NoOpHook)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not subscribe to event processing queue: %w", err)
	}

	cleanup := func() error {
		cancel()
		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup event processing queue: %w", err)
		}
		return nil
	}

	return cleanup, nil
}

func (ec *EventsControllerImpl) handleTask(ctx context.Context, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpanWithCarrier(ctx, "process-event", task.OtelCarrier)
	defer span.End()

	payload := tasktypes.EventTaskPayload{}
	metadata := tasktypes.EventTaskMetadata{}

	err := ec.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode task payload: %w", err)
	}

	err = ec.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode task metadata: %w", err)
	}

	var additionalMetadata map[string]interface{}

	if payload.EventAdditionalMetadata != "" {
		err = json.Unmarshal([]byte(payload.EventAdditionalMetadata), &additionalMetadata)

		if err != nil {
			return fmt.Errorf("could not unmarshal additional metadata: %w", err)
		}
	}

	return ec.processEvent(ctx, metadata.TenantId, payload.EventId, payload.EventKey, []byte(payload.EventData), additionalMetadata)
}

func cleanAdditionalMetadata(additionalMetadata map[string]interface{}) map[string]interface{} {
	if additionalMetadata == nil {
		additionalMetadata = make(map[string]interface{})
	}

	for key := range additionalMetadata {
		if strings.HasPrefix(key, "hatchet__") {
			delete(additionalMetadata, key)
		}
	}
	return additionalMetadata
}

func (ec *EventsControllerImpl) processEvent(ctx context.Context, tenantId, eventId, eventKey string, data []byte, additionalMetadata map[string]interface{}) error {
	additionalMetadata = cleanAdditionalMetadata(additionalMetadata)

	additionalMetadata["hatchet__event_id"] = eventId

	additionalMetadata["hatchet__event_key"] = eventKey

	// query for matching workflows in the system
	workflowVersions, err := ec.repo.Workflow().ListWorkflowsForEvent(ctx, tenantId, eventKey)

	if err != nil {
		return fmt.Errorf("could not query workflows for event: %w", err)
	}

	// create a new workflow run in the database
	var g = new(errgroup.Group)

	for _, workflowVersion := range workflowVersions {
		workflowCp := workflowVersion

		g.Go(func() error {

			// create a new workflow run in the database
			createOpts, err := repository.GetCreateWorkflowRunOptsFromEvent(eventId, workflowCp, data, additionalMetadata)

			if err != nil {
				return fmt.Errorf("could not get create workflow run opts: %w", err)
			}

			workflowRun, err := ec.repo.WorkflowRun().CreateNewWorkflowRun(ctx, tenantId, createOpts)

			if err != nil {
				return fmt.Errorf("processEvent: could not create workflow run: %w", err)
			}

			workflowRunId := sqlchelpers.UUIDToStr(workflowRun.ID)

			// send to workflow processing queue
			err = ec.mq.AddMessage(
				ctx,
				msgqueue.WORKFLOW_PROCESSING_QUEUE,
				tasktypes.WorkflowRunQueuedToTask(
					tenantId,
					workflowRunId,
				),
			)

			if err != nil {
				return fmt.Errorf("could not add workflow run queued task: %w", err)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
