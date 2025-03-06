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
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (i *IngestorImpl) Push(ctx context.Context, req *contracts.PushEventRequest) (*contracts.Event, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	var additionalMeta []byte

	if req.AdditionalMetadata != nil {
		additionalMeta = []byte(*req.AdditionalMetadata)
	}
	event, err := i.IngestEvent(ctx, tenant, req.Key, []byte(req.Payload), additionalMeta)

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

func (i *IngestorImpl) BulkPush(ctx context.Context, req *contracts.BulkPushEventRequest) (*contracts.Events, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	if len(req.Events) == 0 {

		return nil, status.Errorf(codes.InvalidArgument, "No events to ingest")
	}

	if len(req.Events) > 1000 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: too many events - %d is over maximum (1000)", len(req.Events))
	}

	events := make([]*repository.CreateEventOpts, 0)

	for _, e := range req.Events {
		var additionalMeta []byte
		if e.AdditionalMetadata != nil {
			additionalMeta = []byte(*e.AdditionalMetadata)
		}
		events = append(events, &repository.CreateEventOpts{
			TenantId:           tenantId,
			Key:                e.Key,
			Data:               []byte(e.Payload),
			AdditionalMetadata: additionalMeta,
		})
	}

	opts := &repository.BulkCreateEventOpts{
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

	if err == metered.ErrResourceExhausted {
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

	}

	return &contracts.Events{Events: contractEvents}, nil
}

func (i *IngestorImpl) ReplaySingleEvent(ctx context.Context, req *contracts.ReplayEventRequest) (*contracts.Event, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	oldEvent, err := i.eventRepository.GetEventForEngine(ctx, tenantId, req.EventId)

	if err != nil {
		return nil, err
	}

	newEvent, err := i.IngestReplayedEvent(ctx, tenant, oldEvent)

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

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return i.putStreamEventV0(ctx, tenant, req)
	case dbsqlc.TenantMajorEngineVersionV1:
		return i.putStreamEventV1(ctx, tenant, req)
	default:
		return nil, status.Errorf(codes.Unimplemented, "RefreshTimeout is not implemented in engine version %s", string(tenant.Version))
	}
}

func (i *IngestorImpl) putStreamEventV0(ctx context.Context, tenant *dbsqlc.Tenant, req *contracts.PutStreamEventRequest) (*contracts.PutStreamEventResponse, error) {
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

	meta, err := i.streamEventRepository.GetStreamEventMeta(ctx, tenantId, req.StepRunId)

	if err != nil {
		return nil, err
	}

	streamEvent, err := i.streamEventRepository.PutStreamEvent(ctx, tenantId, &opts)

	if err != nil {
		return nil, err
	}

	q := msgqueue.TenantEventConsumerQueue(tenantId)

	e := streamEventToTask(streamEvent, sqlchelpers.UUIDToStr(meta.WorkflowRunId), &meta.RetryCount, &meta.Retries)

	err = i.mq.AddMessage(context.Background(), q, e)

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

	// Make sure we are writing to a step run owned by this tenant
	if t, ok := i.steprunTenantLookupCache.Get(opts.StepRunId); ok {
		if t != tenantId {
			return nil, status.Errorf(codes.PermissionDenied, "Permission denied: step run does not belong to tenant")
		}
		// cache hit
	} else {
		if _, err := i.stepRunRepository.GetStepRunForEngine(ctx, tenantId, opts.StepRunId); err != nil {
			return nil, err
		}

		i.steprunTenantLookupCache.Add(opts.StepRunId, tenantId)
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

func streamEventToTask(e *dbsqlc.StreamEvent, workflowRunId string, retryCount *int32, retries *int32) *msgqueue.Message {
	tenantId := sqlchelpers.UUIDToStr(e.TenantId)

	payloadTyped := tasktypes.StepRunStreamEventTaskPayload{
		WorkflowRunId: workflowRunId,
		StepRunId:     sqlchelpers.UUIDToStr(e.StepRunId),
		CreatedAt:     e.CreatedAt.Time.String(),
		StreamEventId: strconv.FormatInt(e.ID, 10),
		RetryCount:    retryCount,
		StepRetries:   retries,
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
