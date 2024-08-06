package workers

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *WorkerService) WorkerGet(ctx echo.Context, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	worker := ctx.Get("worker").(*db.WorkerModel)

	slotState, recent, err := t.config.APIRepository.Worker().ListWorkerState(worker.TenantID, worker.ID)

	if err != nil {
		return nil, err
	}

	respStepRuns := make([]gen.RecentStepRuns, len(recent))

	for i := range recent {
		genStepRun, err := transformers.ToRecentStepRun(recent[i])

		if err != nil {
			return nil, err
		}

		respStepRuns[i] = *genStepRun
	}

	workerResp := *transformers.ToWorker(worker)

	workerResp.RecentStepRuns = &respStepRuns
	workerResp.Slots = transformers.ToSlotState(slotState)

	affinity, err := t.config.APIRepository.Worker().ListWorkerLabels(worker.TenantID, worker.ID)

	if err != nil {
		return nil, err
	}

	workerResp.Labels = transformers.ToWorkerLabels(affinity)

	return gen.WorkerGet200JSONResponse(workerResp), nil
}
