package transformers

import (
	"time"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
)

func ToWorkflowRuns(
	wfs []*olap.WorkflowRun,
) gen.V2WorkflowRuns {
	toReturn := make([]gen.V2WorkflowRun, len(wfs))

	for i, wf := range wfs {
		toReturn[i] = gen.V2WorkflowRun{
			AdditionalMetadata: wf.AdditionalMetadata,
			DisplayName:        wf.DisplayName,
			Duration:           wf.Duration,
			ErrorMessage:       wf.ErrorMessage,
			FinishedAt:         wf.FinishedAt,
			Id:                 wf.Id,
			Metadata: gen.APIResourceMeta{
				Id:        "1",
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
