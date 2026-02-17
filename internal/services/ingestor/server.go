package ingestor

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	grpcmiddleware "github.com/hatchet-dev/hatchet/pkg/grpc/middleware"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (i *IngestorImpl) Push(ctx context.Context, req *contracts.PushEventRequest) (*contracts.Event, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)

	var additionalMeta []byte

	if req.AdditionalMetadata != nil {
		additionalMeta = []byte(*req.AdditionalMetadata)
	}

	if err := v1.ValidateJSONB(additionalMeta, "additionalMetadata"); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
	}

	payloadBytes := []byte(req.Payload)

	if err := v1.ValidateJSONB(payloadBytes, "payload"); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
	}

	if req.Priority != nil && (*req.Priority < 1 || *req.Priority > 3) {
		return nil, status.Errorf(codes.InvalidArgument, "priority must be between 1 and 3, got %d", *req.Priority)
	}

	event, err := i.IngestEvent(ctx, tenant, req.Key, []byte(req.Payload), additionalMeta, req.Priority, req.Scope, nil)

	if err == v1.ErrResourceExhausted {
		return nil, status.Errorf(codes.ResourceExhausted, "resource exhausted: event limit exceeded for tenant")
	}

	if err != nil {
		return nil, err
	}

	e, err := toEvent(event)

	if err != nil {
		return nil, err
	}

	var additionalMetaStr string

	if req.AdditionalMetadata != nil {
		additionalMetaStr = *req.AdditionalMetadata
	}

	corrId := datautils.ExtractCorrelationId(additionalMetaStr)

	if corrId != nil {
		ctx = context.WithValue(ctx, constants.CorrelationIdKey, *corrId)
	}

	ctx = context.WithValue(ctx, constants.ResourceIdKey, event.ID.String())
	ctx = context.WithValue(ctx, constants.ResourceTypeKey, constants.ResourceTypeEvent)

	grpcmiddleware.TriggerCallback(ctx)

	return e, nil
}

func (i *IngestorImpl) BulkPush(ctx context.Context, req *contracts.BulkPushEventRequest) (*contracts.Events, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)

	tenantId := tenant.ID

	if len(req.Events) == 0 {

		return nil, status.Errorf(codes.InvalidArgument, "No events to ingest")
	}

	if len(req.Events) > 1000 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: too many events - %d is over maximum (1000)", len(req.Events))
	}

	events := make([]*CreateEventOpts, 0)

	for _, e := range req.Events {
		var additionalMeta []byte
		if e.AdditionalMetadata != nil {
			additionalMeta = []byte(*e.AdditionalMetadata)
		}

		if err := v1.ValidateJSONB(additionalMeta, "additionalMetadata"); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
		}

		payloadBytes := []byte(e.Payload)

		if err := v1.ValidateJSONB(payloadBytes, "payload"); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
		}

		if e.Priority != nil && (*e.Priority < 1 || *e.Priority > 3) {
			return nil, status.Errorf(codes.InvalidArgument, "priority must be between 1 and 3, got %d", *e.Priority)
		}

		events = append(events, &CreateEventOpts{
			TenantId:           tenantId,
			Key:                e.Key,
			Data:               payloadBytes,
			AdditionalMetadata: additionalMeta,
			Priority:           e.Priority,
			Scope:              e.Scope,
		})
	}

	opts := &BulkCreateEventOpts{
		TenantId: tenantId,
		Events:   events,
	}

	if err := i.v.Validate(opts); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
	}

	for _, e := range opts.Events {

		if err := i.v.Validate(e); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid request: events failing validation %s", err)
		}
	}

	createdEvents, err := i.BulkIngestEvent(ctx, tenant, events)

	if err == v1.ErrResourceExhausted {
		return nil, status.Errorf(codes.ResourceExhausted, "resource exhausted: event limit exceeded for tenant")
	}
	if err != nil {
		return nil, err
	}

	var contractEvents []*contracts.Event
	for _, e := range createdEvents {

		contractEvent, err := toEvent(e)

		if err != nil {
			return nil, err
		}

		contractEvents = append(contractEvents, contractEvent)

		var additionalMetaStr string

		if e.AdditionalMetadata != nil {
			additionalMetaStr = string(e.AdditionalMetadata)
		}

		corrId := datautils.ExtractCorrelationId(additionalMetaStr)

		if corrId != nil {
			ctx = context.WithValue(ctx, constants.CorrelationIdKey, *corrId)
		}

		ctx = context.WithValue(ctx, constants.ResourceIdKey, e.ID.String())
		ctx = context.WithValue(ctx, constants.ResourceTypeKey, constants.ResourceTypeEvent)

		grpcmiddleware.TriggerCallback(ctx)

	}

	return &contracts.Events{Events: contractEvents}, nil
}

func (i *IngestorImpl) ReplaySingleEvent(ctx context.Context, req *contracts.ReplayEventRequest) (*contracts.Event, error) {
	return nil, status.Errorf(codes.Unimplemented, "ReplaySingleEvent is not implemented")
}

func (i *IngestorImpl) PutStreamEvent(ctx context.Context, req *contracts.PutStreamEventRequest) (*contracts.PutStreamEventResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	return i.putStreamEventV1(ctx, tenant, req)
}

func (i *IngestorImpl) PutLog(ctx context.Context, req *contracts.PutLogRequest) (*contracts.PutLogResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	return i.putLogV1(ctx, tenant, req)
}

func toEvent(e *sqlcv1.Event) (*contracts.Event, error) {
	tenantId := e.TenantId.String()
	eventId := e.ID.String()

	var additionalMeta *string

	if e.AdditionalMetadata != nil {
		additionalMetaStr := string(e.AdditionalMetadata)
		additionalMeta = &additionalMetaStr
	}

	return &contracts.Event{
		TenantId:           tenantId,
		EventId:            eventId,
		Key:                e.Key,
		Payload:            string(e.Data),
		EventTimestamp:     timestamppb.New(e.CreatedAt.Time),
		AdditionalMetadata: additionalMeta,
	}, nil
}

type BulkCreateEventOpts struct {
	TenantId uuid.UUID `validate:"required"`
	Events   []*CreateEventOpts
}

type CreateEventOpts struct {
	ReplayedEvent         *uuid.UUID `validate:"omitempty"`
	Priority              *int32     `validate:"omitempty,min=1,max=3"`
	Scope                 *string    `validate:"omitempty"`
	TriggeringWebhookName *string    `validate:"omitempty"`
	TenantId              uuid.UUID  `validate:"required"`
	Key                   string     `validate:"required"`
	Data                  []byte
	AdditionalMetadata    []byte
}
