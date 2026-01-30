package transformers

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
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
		TenantId: pgUUIDToStr(event.TenantId),
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
		Metadata:           *toAPIMetadata(pgUUIDToStr(event.EventExternalID), event.EventSeenAt.Time, event.EventSeenAt.Time),
		Key:                event.EventKey,
		TenantId:           pgUUIDToStr(event.TenantID),
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

func pgUUIDToStr(uuid uuid.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid.Bytes[0:4], uuid.Bytes[4:6], uuid.Bytes[6:8], uuid.Bytes[8:10], uuid.Bytes[10:16])
}
