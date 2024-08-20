package workers

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *WorkerService) WorkerGet(ctx echo.Context, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	worker := ctx.Get("worker").(*dbsqlc.GetWorkerByIdRow)

	recentFailFilter := false

	if request.Params.RecentFailed != nil {
		recentFailFilter = *request.Params.RecentFailed
	}

	slotState, recent, err := t.config.APIRepository.Worker().ListWorkerState(
		sqlchelpers.UUIDToStr(worker.Worker.TenantId),
		sqlchelpers.UUIDToStr(worker.Worker.ID),
		recentFailFilter)

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

	workerResp := *transformers.ToWorkerSqlc(&worker.Worker, nil, &worker.WebhookUrl.String)

	workerResp.RecentStepRuns = &respStepRuns
	workerResp.Slots = transformers.ToSlotState(slotState)

	affinity, err := t.config.APIRepository.Worker().ListWorkerLabels(
		sqlchelpers.UUIDToStr(worker.Worker.TenantId),
		sqlchelpers.UUIDToStr(worker.Worker.ID),
	)

	if err != nil {
		return nil, err
	}

	workerResp.Labels = transformers.ToWorkerLabels(affinity)

	return gen.WorkerGet200JSONResponse(workerResp), nil
}
