package workers

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (t *WorkerService) WorkerGet(ctx echo.Context, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	worker := ctx.Get("worker").(*db.WorkerModel)

	stepRuns, err := t.config.APIRepository.Worker().ListRecentWorkerStepRuns(worker.TenantID, worker.ID)

	if err != nil {
		return nil, err
	}

	respStepRuns := make([]gen.StepRun, len(stepRuns))

	for i := range stepRuns {
		genStepRun, err := transformers.ToStepRun(&stepRuns[i])

		if err != nil {
			return nil, err
		}

		respStepRuns[i] = *genStepRun
	}

	workerResp := *transformers.ToWorker(worker)

	workerResp.RecentStepRuns = &respStepRuns

	return gen.WorkerGet200JSONResponse(workerResp), nil
}
