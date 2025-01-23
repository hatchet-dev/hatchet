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
			Id:           wf.Id,
			ErrorMessage: &wf.ErrorMessage,
			Metadata: gen.APIResourceMeta{
				Id:        "1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Status:    wf.Status,
			TaskId:    wf.TaskId,
			Timestamp: wf.Timestamp,
		}
	}

	return gen.V2WorkflowRuns{
		WorkflowRuns: toReturn,
		Metadata: gen.APIResourceMeta{
			Id:        "1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}
