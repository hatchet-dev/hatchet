package transformers

import (
	"encoding/json"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToEventList(events []*sqlcv1.Event) []gen.Event {
	res := make([]gen.Event, len(events))

	for i, event := range events {
		res[i] = ToEvent(event)
	}

	return res
}

func ToEvent(event *sqlcv1.Event) gen.Event {
	return gen.Event{
		Metadata: *toAPIMetadata(sqlchelpers.UUIDToStr(event.ID), event.CreatedAt.Time, event.UpdatedAt.Time),
		Key:      event.Key,
		TenantId: event.TenantId.String(),
	}
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
		Metadata:           *toAPIMetadata(event.EventExternalID.String(), event.EventSeenAt.Time, event.EventSeenAt.Time),
		Key:                event.EventKey,
		TenantId:           event.TenantID.String(),
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
