package transformers

import (
	"encoding/json"
	"time"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
)

func jsonToMap(jsonStr string) map[string]interface{} {
	result := make(map[string]interface{})
	json.Unmarshal([]byte(jsonStr), &result)
	return result
}

func ToWorkflowRuns(
	wfs []*olap.WorkflowRun,
) gen.V2WorkflowRuns {
	toReturn := make([]gen.V2WorkflowRun, len(wfs))

	for i, wf := range wfs {
		additionalMetadata := jsonToMap(*wf.AdditionalMetadata)
		var duration *int
		if wf.Duration != nil {
			d := int(*wf.Duration)
			duration = &d
		}

		toReturn[i] = gen.V2WorkflowRun{
			AdditionalMetadata: &additionalMetadata,
			DisplayName:        wf.DisplayName,
			Duration:           duration,
			ErrorMessage:       wf.ErrorMessage,
			FinishedAt:         wf.FinishedAt,
			Id:                 wf.Id,
			Metadata: gen.APIResourceMeta{
				Id:        wf.TaskId.String(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			StartedAt: wf.StartedAt,
			Status:    wf.Status,
			TaskId:    wf.TaskId,
			TenantId:  wf.TenantId,
			Timestamp: wf.Timestamp,
		}
	}

	return gen.V2WorkflowRuns{
		Rows:       toReturn,
		Pagination: gen.PaginationResponse{},
	}
}
