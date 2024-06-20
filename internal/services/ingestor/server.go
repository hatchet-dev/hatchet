package ingestor

import (
	"context"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (i *IngestorImpl) Push(ctx context.Context, req *contracts.PushEventRequest) (*contracts.Event, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	var additionalMeta []byte

	if req.AdditionalMetadata != nil {
		additionalMeta = []byte(*req.AdditionalMetadata)
	}
	event, err := i.IngestEvent(ctx, tenantId, req.Key, []byte(req.Payload), additionalMeta)

	if err == metered.ErrResourceExhausted {
		return nil, status.Errorf(codes.ResourceExhausted, "resource exhausted: event limit exceeded for tenant")
	}

	if err != nil {
		return nil, err
	}

	e, err := toEvent(event)

	if err != nil {
		return nil, err
	}

	return e, nil
}

func (i *IngestorImpl) ReplaySingleEvent(ctx context.Context, req *contracts.ReplayEventRequest) (*contracts.Event, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	oldEvent, err := i.eventRepository.GetEventForEngine(ctx, tenantId, req.EventId)

	if err != nil {
		return nil, err
	}

	newEvent, err := i.IngestReplayedEvent(ctx, tenantId, oldEvent)

	if err != nil {
		return nil, err
	}

	e, err := toEvent(newEvent)

	if err != nil {
		return nil, err
	}

	return e, nil
}

func (i *IngestorImpl) PutStreamEvent(ctx context.Context, req *contracts.PutStreamEventRequest) (*contracts.PutStreamEventResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	var createdAt *time.Time

	if t := req.CreatedAt.AsTime().UTC(); !t.IsZero() {
		createdAt = &t
	}

	var metadata []byte

	if req.Metadata != "" {
		metadata = []byte(req.Metadata)
	}

	opts := repository.CreateStreamEventOpts{
		StepRunId: req.StepRunId,
		CreatedAt: createdAt,
		Message:   req.Message,
		Metadata:  metadata,
	}

	if err := i.v.Validate(opts); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
	}

	streamEvent, err := i.streamEventRepository.PutStreamEvent(ctx, tenantId, &opts)

	if err != nil {
		return nil, err
	}

	q, err := msgqueue.TenantEventConsumerQueue(tenantId)

	if err != nil {
		return nil, err
	}

	err = i.mq.AddMessage(context.Background(), q, streamEventToTask(streamEvent))

	if err != nil {
		return nil, err
	}

	return &contracts.PutStreamEventResponse{}, nil
}

func (i *IngestorImpl) PutLog(ctx context.Context, req *contracts.PutLogRequest) (*contracts.PutLogResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	var createdAt *time.Time

	if t := req.CreatedAt.AsTime(); !t.IsZero() {
		createdAt = &t
	}

	var metadata []byte

	if req.Metadata != "" {
		metadata = []byte(req.Metadata)
	}

	opts := &repository.CreateLogLineOpts{
		StepRunId: req.StepRunId,
		CreatedAt: createdAt,
		Message:   req.Message,
		Level:     req.Level,
		Metadata:  metadata,
	}

	if apiErrors, err := i.v.ValidateAPI(opts); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", apiErrors.String())
	}

	_, err := i.logRepository.PutLog(ctx, tenantId, opts)

	if err != nil {
		return nil, err
	}

	return &contracts.PutLogResponse{}, nil
}

func toEvent(e *dbsqlc.Event) (*contracts.Event, error) {
	tenantId := sqlchelpers.UUIDToStr(e.TenantId)
	eventId := sqlchelpers.UUIDToStr(e.ID)

	return &contracts.Event{
		TenantId:       tenantId,
		EventId:        eventId,
		Key:            e.Key,
		Payload:        string(e.Data),
		EventTimestamp: timestamppb.New(e.CreatedAt.Time),
	}, nil
}

func streamEventToTask(e *dbsqlc.StreamEvent) *msgqueue.Message {
	tenantId := sqlchelpers.UUIDToStr(e.TenantId)

	payloadTyped := tasktypes.StepRunStreamEventTaskPayload{
		StepRunId:     sqlchelpers.UUIDToStr(e.StepRunId),
		CreatedAt:     e.CreatedAt.Time.String(),
		StreamEventId: strconv.FormatInt(e.ID, 10),
	}

	payload, _ := datautils.ToJSONMap(payloadTyped)

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunStreamEventTaskMetadata{
		TenantId:      tenantId,
		StreamEventId: strconv.FormatInt(e.ID, 10),
	})

	return &msgqueue.Message{
		ID:       "step-run-stream-event",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}
