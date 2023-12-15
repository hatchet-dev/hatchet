package workers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/labstack/echo/v4"
)

func (t *WorkerService) WorkerGet(ctx echo.Context, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	worker := ctx.Get("worker").(*db.WorkerModel)

	stepRuns, err := t.config.Repository.Worker().ListRecentWorkerStepRuns(worker.TenantID, worker.ID)

	if err != nil {
		return nil, err
	}

	respStepRuns := make([]gen.StepRun, len(stepRuns))

	for i := range stepRuns {
		respStepRuns[i] = *transformers.ToStepRun(&stepRuns[i])
	}

	workerResp := *transformers.ToWorker(worker)

	workerResp.RecentStepRuns = &respStepRuns

	return gen.WorkerGet200JSONResponse(workerResp), nil
}
