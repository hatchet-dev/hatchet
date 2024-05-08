package transformers

import (
	"encoding/json"
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

func ToEvent(event *db.EventModel) *gen.Event {
	res := &gen.Event{
		Metadata: *toAPIMetadata(event.ID, event.CreatedAt, event.UpdatedAt),
		Key:      event.Key,
		TenantId: event.TenantID,
	}

	return res
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
		Metadata:           *toAPIMetadata(pgUUIDToStr(event.ID), event.CreatedAt.Time, event.UpdatedAt.Time),
		Key:                event.Key,
		TenantId:           pgUUIDToStr(event.TenantId),
		AdditionalMetadata: &metadata,
	}

	res.WorkflowRunSummary = &gen.EventWorkflowRunSummary{
		Failed:    &eventRow.Failedruns,
		Running:   &eventRow.Runningruns,
		Succeeded: &eventRow.Succeededruns,
		Pending:   &eventRow.Pendingruns,
		Queued:    &eventRow.Queuedruns,
	}

	return res, nil
}

func pgUUIDToStr(uuid pgtype.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid.Bytes[0:4], uuid.Bytes[4:6], uuid.Bytes[6:8], uuid.Bytes[8:10], uuid.Bytes[10:16])
}
