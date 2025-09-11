package ingestor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	schedulingv1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
)

type EventResult struct {
	TenantId           string
	EventId            string
	EventKey           string
	Data               string
	AdditionalMetadata string
}

func (i *IngestorImpl) ingestEventV1(ctx context.Context, tenant *dbsqlc.Tenant, key string, data []byte, metadata []byte, priority *int32, scope, triggeringWebhookName *string) (*dbsqlc.Event, error) {
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

	opt := eventToPayload(tenantId, key, data, metadata, priority, scope, triggeringWebhookName)

	events, err := i.ingest(ctx, tenant, opt)

	if err != nil {
		return nil, fmt.Errorf("could not ingest event: %w", err)
	}

	if len(events) != 1 {
		return nil, fmt.Errorf("expected 1 event to be ingested, got %d", len(events))
	}

	return events[0], nil
}

func (i *IngestorImpl) ingest(ctx context.Context, tenant *dbsqlc.Tenant, eventOpts ...tasktypes.UserEventTaskPayload) ([]*dbsqlc.Event, error) {
	res := make([]*dbsqlc.Event, 0, len(eventOpts))
	now := time.Now().UTC()
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	for _, event := range eventOpts {
		e := &dbsqlc.Event{
			ID:                 sqlchelpers.UUIDFromStr(event.EventExternalId),
			CreatedAt:          sqlchelpers.TimestampFromTime(now),
			UpdatedAt:          sqlchelpers.TimestampFromTime(now),
			Key:                event.EventKey,
			TenantId:           sqlchelpers.UUIDFromStr(tenantId),
			Data:               event.EventData,
			AdditionalMetadata: event.EventAdditionalMetadata,
		}

		res = append(res, e)
	}

	wasProcessedLocally := false

	if i.localScheduler != nil {
		localWorkerIds := map[string]struct{}{}

		if i.localDispatcher != nil {
			localWorkerIds = i.localDispatcher.GetLocalWorkerIds()
		}

		opts := make([]v1.EventTriggerOpts, 0)

		for _, event := range eventOpts {
			opts = append(opts, v1.EventTriggerOpts{
				ExternalId:            event.EventExternalId,
				Key:                   event.EventKey,
				Data:                  event.EventData,
				AdditionalMetadata:    event.EventAdditionalMetadata,
				Priority:              event.EventPriority,
				Scope:                 event.EventScope,
				TriggeringWebhookName: event.TriggeringWebhookName,
			})
		}

		localAssigned, schedulingErr := i.localScheduler.RunOptimisticSchedulingFromEvents(ctx, tenantId, opts, localWorkerIds)

		if schedulingErr != nil {
			if !errors.Is(schedulingErr, schedulingv1.ErrTenantNotFound) && !errors.Is(schedulingErr, schedulingv1.ErrNoOptimisticSlots) {
				i.l.Error().Err(schedulingErr).Msg("could not run optimistic scheduling")
			}
		}

		if i.localDispatcher != nil && len(localAssigned) > 0 {
			eg := errgroup.Group{}

			for workerId, assignedItems := range localAssigned {
				eg.Go(func() error {
					err := i.localDispatcher.HandleLocalAssignments(ctx, tenantId, workerId, assignedItems)

					if err != nil {
						return fmt.Errorf("could not dispatch assigned items: %w", err)
					}

					return nil
				})
			}

			dispatcherErr := eg.Wait()

			if dispatcherErr != nil {
				i.l.Error().Err(dispatcherErr).Msg("could not handle local assignments")
			}
		}

		// if there's no scheduling error, the event was processed locally. Note that we don't return here because
		// we still need to enqueue the event to ensure downstream processing (triggers, durable events)
		if schedulingErr == nil {
			wasProcessedLocally = true
		}
	}

	ctx, span := telemetry.NewSpan(ctx, "ingest-events")
	defer span.End()

	var outerErr error

	for _, event := range eventOpts {
		event.WasProcessedLocally = wasProcessedLocally

		msg, err := msgqueue.NewTenantMessage(
			tenantId,
			"user-event",
			false,
			true,
			event,
		)

		if err != nil {
			outerErr = multierror.Append(outerErr, fmt.Errorf("could not create event task: %w", err))
			continue
		}

		err = i.mqv1.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)

		if err != nil {
			outerErr = multierror.Append(outerErr, fmt.Errorf("could not add event to task queue: %w", err))
		}
	}

	if outerErr != nil {
		return nil, fmt.Errorf("failed to ingest events: %w", outerErr)
	}

	return res, nil
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

	payloads := make([]tasktypes.UserEventTaskPayload, 0, len(eventOpts))

	for _, event := range eventOpts {
		payloads = append(payloads, eventToPayload(tenantId, event.Key, event.Data, event.AdditionalMetadata, event.Priority, event.Scope, event.TriggeringWebhookName))
	}

	return i.ingest(ctx, tenant, payloads...)
}

func (i *IngestorImpl) ingestReplayedEventV1(ctx context.Context, tenant *dbsqlc.Tenant, replayedEvent *dbsqlc.Event) (*dbsqlc.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "ingest-replayed-event")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	opt := eventToPayload(tenantId, replayedEvent.Key, replayedEvent.Data, replayedEvent.AdditionalMetadata, nil, nil, nil)

	events, err := i.ingest(ctx, tenant, opt)

	if err != nil {
		return nil, fmt.Errorf("could not ingest event: %w", err)
	}

	if len(events) != 1 {
		return nil, fmt.Errorf("expected 1 event to be ingested, got %d", len(events))
	}

	return events[0], nil
}

func eventToPayload(tenantId, key string, data, additionalMeta []byte, priority *int32, scope *string, triggeringWebhookName *string) tasktypes.UserEventTaskPayload {
	eventId := uuid.New().String()

	return tasktypes.UserEventTaskPayload{
		EventExternalId:         eventId,
		EventKey:                key,
		EventData:               data,
		EventAdditionalMetadata: additionalMeta,
		EventPriority:           priority,
		EventScope:              scope,
		TriggeringWebhookName:   triggeringWebhookName,
	}
}

func createWebhookValidationFailureMsg(tenantId, webhookName, errorText string) (*msgqueue.Message, error) {
	payloadTyped := tasktypes.FailedWebhookValidationPayload{
		WebhookName: webhookName,
		ErrorText:   errorText,
	}

	return msgqueue.NewTenantMessage(
		tenantId,
		"failed-webhook-validation",
		false,
		true,
		payloadTyped,
	)
}

func (i *IngestorImpl) ingestWebhookValidationFailure(tenantId, webhookName, errorText string) error {
	msg, err := createWebhookValidationFailureMsg(
		tenantId,
		webhookName,
		errorText,
	)

	if err != nil {
		return fmt.Errorf("could not create failed webhook validation payload: %w", err)
	}

	err = i.mqv1.SendMessage(context.Background(), msgqueue.OLAP_QUEUE, msg)

	if err != nil {
		return fmt.Errorf("could not add failed webhook validation to olap queue: %w", err)
	}

	return nil
}

func (i *IngestorImpl) ingestCELEvaluationFailure(ctx context.Context, tenantId, errorText string, source sqlcv1.V1CelEvaluationFailureSource) error {
	msg, err := tasktypes.CELEvaluationFailureMessage(
		tenantId,
		[]v1.CELEvaluationFailure{
			{
				Source:       source,
				ErrorMessage: errorText,
			},
		},
	)

	if err != nil {
		return fmt.Errorf("failed to create CEL evaluation failure message: %w", err)
	}

	err = i.mqv1.SendMessage(ctx, msgqueue.OLAP_QUEUE, msg)

	if err != nil {
		return fmt.Errorf("failed to send CEL evaluation failure message: %w", err)
	}

	return nil
}
