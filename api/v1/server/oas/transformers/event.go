package transformers

import (
	"encoding/json"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
)

func ToEventList(events []*dbsqlc.Event) []gen.Event {
	res := make([]gen.Event, len(events))

	for i, event := range events {
		res[i] = ToEvent(event)
	}

	return res
}

func ToEvent(event *dbsqlc.Event) gen.Event {
	return gen.Event{
		Metadata: *toAPIMetadata(sqlchelpers.UUIDToStr(event.ID), event.CreatedAt.Time, event.UpdatedAt.Time),
		Key:      event.Key,
		TenantId: sqlchelpers.UUIDToStr(event.TenantId),
	}
}

func ToEventFromSQLC(eventRow *dbsqlc.ListEventsRow) (*gen.Event, error) {
	event := eventRow.Event

	var metadata map[string]interface{}

	if event.AdditionalMetadata != nil {
		err := json.Unmarshal(event.AdditionalMetadata, &metadata)
		if err != nil {
			return nil, err
		}
	}

	res := &gen.Event{
		Metadata:           *toAPIMetadata(sqlchelpers.UUIDToStr(event.ID), event.CreatedAt.Time, event.UpdatedAt.Time),
		Key:                event.Key,
		TenantId:           sqlchelpers.UUIDToStr(event.TenantId),
		AdditionalMetadata: &metadata,
	}

	res.WorkflowRunSummary = &gen.EventWorkflowRunSummary{
		Failed:    &eventRow.Failedruns,
		Running:   &eventRow.Runningruns,
		Succeeded: &eventRow.Succeededruns,
		Pending:   &eventRow.Pendingruns,
		Queued:    &eventRow.Queuedruns,
		Cancelled: &eventRow.Cancelledruns,
	}

	return res, nil
}

func ToEventFromSQLCV1(event *v1.EventWithPayload) (*gen.Event, error) {
	var metadata map[string]interface{}

	if event.EventAdditionalMetadata != nil {
		err := json.Unmarshal(event.EventAdditionalMetadata, &metadata)
		if err != nil {
			return nil, err
		}
	}

	res := &gen.Event{
		Metadata:           *toAPIMetadata(sqlchelpers.UUIDToStr(event.EventExternalID), event.EventSeenAt.Time, event.EventSeenAt.Time),
		Key:                event.EventKey,
		TenantId:           sqlchelpers.UUIDToStr(event.TenantID),
		AdditionalMetadata: &metadata,
	}

	res.WorkflowRunSummary = &gen.EventWorkflowRunSummary{
		Failed:    &event.FailedCount,
		Running:   &event.RunningCount,
		Succeeded: &event.CompletedCount,
		Pending:   &event.QueuedCount,
		Queued:    &event.QueuedCount,
		Cancelled: &event.CancelledCount,
	}

	return res, nil
}
