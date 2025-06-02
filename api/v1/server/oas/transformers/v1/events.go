package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func ToV1EventList(events []*sqlcv1.ListEventsRow, limit, offset, total int64) gen.V1EventList {
	rows := make([]gen.V1Event, len(events))

	numPages := (total / limit) + 1
	currentPage := (offset / limit) + 1

	var nextPage int64
	if total < offset+limit {
		nextPage = currentPage
	} else {
		nextPage = currentPage + 1
	}

	pagination := gen.PaginationResponse{
		CurrentPage: &currentPage,
		NextPage:    &nextPage,
		NumPages:    &numPages,
	}

	for i, row := range events {
		additionalMetadata := jsonToMap(row.EventAdditionalMetadata)

		rows[i] = gen.V1Event{
			AdditionalMetadata: &additionalMetadata,
			Key:                row.EventKey,
			Metadata: gen.APIResourceMeta{
				CreatedAt: row.EventSeenAt.Time,
				UpdatedAt: row.EventSeenAt.Time,
				Id:        row.EventExternalID.String(),
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
