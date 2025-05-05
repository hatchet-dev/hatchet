package transformers

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func ToV1EventList(events []*sqlcv1.ListEventsRow) gen.V1EventList {
	rows := make([]gen.V1Event, len(events))

	pagination := gen.PaginationResponse{}

	for i, row := range events {
		additionalMetadata := jsonToMap(row.EventAdditionalMetadata)
		externalId := uuid.MustParse(sqlchelpers.UUIDToStr(row.EventExternalID))

		rows[i] = gen.V1Event{
			AdditionalMetadata: &additionalMetadata,
			Key:                row.EventKey,
			Metadata: gen.APIResourceMeta{
				CreatedAt: row.EventGeneratedAt.Time,
				UpdatedAt: row.EventGeneratedAt.Time,
				Id:        strconv.FormatInt(row.EventID, 10),
			},
			WorkflowRunSummary: gen.V1EventWorkflowRunSummary{
				Cancelled: row.CancelledCount.Int64,
				Succeeded: row.CompletedCount.Int64,
				Queued:    row.QueuedCount.Int64,
				Failed:    row.FailedCount.Int64,
				Running:   row.RunningCount.Int64,
			},
			ExternalId: externalId,
		}
	}

	return gen.V1EventList{
		Rows:       &rows,
		Pagination: &pagination,
	}
}
