package workers

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkerService) WorkerGet(ctx echo.Context, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	worker := ctx.Get("worker").(*dbsqlc.GetWorkerByIdRow)

	slotState, recent, err := t.config.APIRepository.Worker().ListWorkerState(
		sqlchelpers.UUIDToStr(worker.Worker.TenantId),
		sqlchelpers.UUIDToStr(worker.Worker.ID),
		int(worker.Worker.MaxRuns),
	)

	if err != nil {
		return nil, err
	}

	actions, err := t.config.APIRepository.Worker().GetWorkerActionsByWorkerId(
		sqlchelpers.UUIDToStr(worker.Worker.TenantId),
		sqlchelpers.UUIDToStr(worker.Worker.ID),
	)

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

	slots := int(worker.RemainingSlots)

	workerResp := *transformers.ToWorkerSqlc(&worker.Worker, &slots, &worker.WebhookUrl.String, actions)

	workerResp.RecentStepRuns = &respStepRuns
	workerResp.Slots = transformers.ToSlotState(slotState, slots)

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
