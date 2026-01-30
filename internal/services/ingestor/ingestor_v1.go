package ingestor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type EventResult struct {
	TenantId           string
	EventId            string
	EventKey           string
	Data               string
	AdditionalMetadata string
}

func (i *IngestorImpl) ingestEventV1(ctx context.Context, tenant *sqlcv1.Tenant, key string, data []byte, metadata []byte, priority *int32, scope, triggeringWebhookName *string) (*sqlcv1.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "ingest-event")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	canCreateEvents, eLimit, err := i.repov1.TenantLimit().CanCreate(
		ctx,
		sqlcv1.LimitResourceEVENT,
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

	return i.ingestSingleton(ctx, tenantId, key, data, metadata, priority, scope, triggeringWebhookName)
}

func (i *IngestorImpl) ingestSingleton(ctx context.Context, tenantId, key string, data []byte, metadata []byte, priority *int32, scope, triggeringWebhookName *string) (*sqlcv1.Event, error) {
	eventId := uuid.New().String()

	msg, err := eventToTaskV1(
		tenantId,
		eventId,
		key,
		data,
		metadata,
		priority,
		scope,
		triggeringWebhookName,
	)

	if err != nil {
		return nil, fmt.Errorf("could not create event task: %w", err)
	}

	err = i.mqv1.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return nil, fmt.Errorf("could not add event to task queue: %w", err)
	}

	now := time.Now().UTC()

	return &sqlcv1.Event{
		ID:                 uuid.MustParse(eventId),
		CreatedAt:          sqlchelpers.TimestampFromTime(now),
		UpdatedAt:          sqlchelpers.TimestampFromTime(now),
		Key:                key,
		TenantId:           uuid.MustParse(tenantId),
		Data:               data,
		AdditionalMetadata: metadata,
	}, nil
}

func (i *IngestorImpl) bulkIngestEventV1(ctx context.Context, tenant *sqlcv1.Tenant, eventOpts []*CreateEventOpts) ([]*sqlcv1.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "bulk-ingest-event")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	count := len(eventOpts)

	canCreateEvents, eLimit, err := i.repov1.TenantLimit().CanCreate(
		ctx,
		sqlcv1.LimitResourceEVENT,
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

	results := make([]*sqlcv1.Event, 0, len(eventOpts))

	for _, event := range eventOpts {
		res, err := i.ingestSingleton(ctx, tenantId, event.Key, event.Data, event.AdditionalMetadata, event.Priority, event.Scope, event.TriggeringWebhookName)

		if err != nil {
			return nil, fmt.Errorf("could not ingest event: %w", err)
		}

		results = append(results, res)
	}

	return results, nil
}

func (i *IngestorImpl) ingestReplayedEventV1(ctx context.Context, tenant *sqlcv1.Tenant, replayedEvent *sqlcv1.Event) (*sqlcv1.Event, error) {
	ctx, span := telemetry.NewSpan(ctx, "ingest-replayed-event")
	defer span.End()

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	return i.ingestSingleton(ctx, tenantId, replayedEvent.Key, replayedEvent.Data, replayedEvent.AdditionalMetadata, nil, nil, nil)
}

func eventToTaskV1(tenantId, eventExternalId, key string, data, additionalMeta []byte, priority *int32, scope *string, triggeringWebhookName *string) (*msgqueue.Message, error) {
	payloadTyped := tasktypes.UserEventTaskPayload{
		EventExternalId:         eventExternalId,
		EventKey:                key,
		EventData:               data,
		EventAdditionalMetadata: additionalMeta,
		EventPriority:           priority,
		EventScope:              scope,
		TriggeringWebhookName:   triggeringWebhookName,
	}

	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDUserEvent,
		false,
		true,
		payloadTyped,
	)
}

func createWebhookValidationFailureMsg(tenantId, webhookName, errorText string) (*msgqueue.Message, error) {
	payloadTyped := tasktypes.FailedWebhookValidationPayload{
		WebhookName: webhookName,
		ErrorText:   errorText,
	}

	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDFailedWebhookValidation,
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
