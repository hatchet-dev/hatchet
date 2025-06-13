package transformers

import (
	"encoding/json"
	"math"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func parseTriggeredRuns(triggeredRuns []byte) ([]gen.V1EventTriggeredRun, error) {
	var result []gen.V1EventTriggeredRun

	if len(triggeredRuns) > 0 {
		var rawTriggeredRuns []map[string]interface{}
		if err := json.Unmarshal(triggeredRuns, &rawTriggeredRuns); err != nil {
			result = []gen.V1EventTriggeredRun{}
		} else {
			result = make([]gen.V1EventTriggeredRun, len(rawTriggeredRuns))
			for j, rawRun := range rawTriggeredRuns {
				var runExternalId uuid.UUID
				var filterId *uuid.UUID

				if runExtIdRaw, ok := rawRun["run_external_id"]; ok && runExtIdRaw != nil {
					if runExtIdStr, ok := runExtIdRaw.(string); ok {
						if parsedUUID, err := uuid.Parse(runExtIdStr); err == nil {
							runExternalId = parsedUUID
						}
					}
				}

				if filterIdRaw, ok := rawRun["filter_id"]; ok && filterIdRaw != nil {
					if filterIdStr, ok := filterIdRaw.(string); ok && filterIdStr != "" {
						if parsedUUID, err := uuid.Parse(filterIdStr); err == nil {
							filterId = &parsedUUID
						}
					}
				}

				result[j] = gen.V1EventTriggeredRun{
					WorkflowRunId: runExternalId,
					FilterId:      filterId,
				}
			}
		}
	}

	return result, nil
}

func ToV1EventList(events []*sqlcv1.ListEventsRow, limit, offset, total int64) gen.V1EventList {
	rows := make([]gen.V1Event, len(events))

	numPages := int64(math.Ceil(float64(total) / float64(limit)))
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
		payload := jsonToMap(row.EventPayload)
		triggeredRuns, err := parseTriggeredRuns(row.TriggeredRuns)

		if err != nil {
			triggeredRuns = []gen.V1EventTriggeredRun{}
		}

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
			Payload:       &payload,
			SeenAt:        &row.EventSeenAt.Time,
			Scope:         &row.EventScope.String,
			TriggeredRuns: &triggeredRuns,
		}
	}

	return gen.V1EventList{
		Rows:       &rows,
		Pagination: &pagination,
	}
}
