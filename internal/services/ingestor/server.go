package ingestor

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
)

func (i *IngestorImpl) Push(ctx context.Context, req *contracts.PushEventRequest) (*contracts.Event, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	event, err := i.IngestEvent(ctx, tenantId, req.Key, []byte(req.Payload))

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

	oldEvent, err := i.eventRepository.GetEventForEngine(tenantId, req.EventId)

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

	_, err := i.logRepository.PutLog(tenantId, &repository.CreateLogLineOpts{
		StepRunId: req.StepRunId,
		CreatedAt: createdAt,
		Message:   req.Message,
		Level:     req.Level,
		Metadata:  metadata,
	})

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
