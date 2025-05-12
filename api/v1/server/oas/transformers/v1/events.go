package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func ToV1EventList(events []*sqlcv1.ListEventsRow) gen.V1EventList {
	rows := make([]gen.V1Event, len(events))

	pagination := gen.PaginationResponse{}

	for i, row := range events {
		additionalMetadata := jsonToMap(row.EventAdditionalMetadata)

		rows[i] = gen.V1Event{
			AdditionalMetadata: &additionalMetadata,
			Key:                row.EventKey,
			Metadata: gen.APIResourceMeta{
				CreatedAt: row.EventSeenAt.Time,
				UpdatedAt: row.EventSeenAt.Time,
				Id:        row.EventID.String(),
			},
			WorkflowRunSummary: gen.V1EventWorkflowRunSummary{
				Cancelled: row.CancelledCount.Int64,
				Succeeded: row.CompletedCount.Int64,
				Queued:    row.QueuedCount.Int64,
				Failed:    row.FailedCount.Int64,
				Running:   row.RunningCount.Int64,
			},
		}
	}

	return gen.V1EventList{
		Rows:       &rows,
		Pagination: &pagination,
	}
}
