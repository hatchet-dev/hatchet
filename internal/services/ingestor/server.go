package ingestor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jackc/pgx/v5/pgtype"
)

func (i *IngestorImpl) Push(ctx context.Context, req *contracts.PushEventRequest) (*contracts.Event, error) {
	eventDataMap := map[string]interface{}{}

	err := json.Unmarshal([]byte(req.Payload), &eventDataMap)

	if err != nil {
		return nil, err
	}

	event, err := i.IngestEvent(req.TenantId, req.Key, eventDataMap)

	if err != nil {
		return nil, err
	}

	e, err := toEvent(*event)

	if err != nil {
		return nil, err
	}

	return e, nil
}

func (i *IngestorImpl) List(ctx context.Context, req *contracts.ListEventRequest) (*contracts.ListEventResponse, error) {
	offset := int(req.Offset)
	var keys []string

	if req.Key != "" {
		keys = []string{req.Key}
	}

	listResult, err := i.eventRepository.ListEvents(req.TenantId, &repository.ListEventOpts{
		Keys:   keys,
		Offset: &offset,
	})

	if err != nil {
		return nil, err
	}

	items := []*contracts.Event{}

	for _, event := range listResult.Rows {
		e, err := toEventFromSQLC(event)

		if err != nil {
			return nil, err
		}

		items = append(items, e)
	}

	return &contracts.ListEventResponse{
		Events: items,
	}, nil
}

func (i *IngestorImpl) ReplaySingleEvent(ctx context.Context, req *contracts.ReplayEventRequest) (*contracts.Event, error) {
	oldEvent, err := i.eventRepository.GetEventById(req.EventId)

	if err != nil {
		return nil, err
	}

	newEvent, err := i.IngestReplayedEvent(req.TenantId, oldEvent)

	if err != nil {
		return nil, err
	}

	e, err := toEvent(*newEvent)

	if err != nil {
		return nil, err
	}

	return e, nil
}

func toEventFromSQLC(eventRow *dbsqlc.ListEventsRow) (*contracts.Event, error) {
	event := eventRow.Event

	var payload string

	if event.Data != nil {
		payload = string(event.Data)
	}

	return &contracts.Event{
		TenantId:       pgUUIDToStr(event.TenantId),
		EventId:        pgUUIDToStr(event.ID),
		Key:            event.Key,
		Payload:        payload,
		EventTimestamp: timestamppb.New(event.CreatedAt.Time),
	}, nil
}

func pgUUIDToStr(uuid pgtype.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid.Bytes[0:4], uuid.Bytes[4:6], uuid.Bytes[6:8], uuid.Bytes[8:10], uuid.Bytes[10:16])
}

func toEvent(e db.EventModel) (*contracts.Event, error) {
	var payload string

	if data, ok := e.Data(); ok {
		payloadBytes, err := data.MarshalJSON()

		if err != nil {
			return nil, err
		}

		payload = string(payloadBytes)
	}

	return &contracts.Event{
		TenantId:       e.TenantID,
		EventId:        e.ID,
		Key:            e.Key,
		Payload:        payload,
		EventTimestamp: timestamppb.New(e.CreatedAt),
	}, nil
}

// func (contracts.UnimplementedEventsServiceServer).List(context.Context, *contracts.ListEventRequest) (*contracts.ListEventResponse, error)
// func (contracts.UnimplementedEventsServiceServer).Push(context.Context, *contracts.Event) (*contracts.EventPushResponse, error)
// func (contracts.UnimplementedEventsServiceServer).ReplaySingleEvent(context.Context, *contracts.ReplayEventRequest) (*contracts.EventPushResponse, error)
