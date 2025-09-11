package workers

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	transformersv1 "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkerService) WorkerGet(ctx echo.Context, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return t.workerGetV0(ctx, tenant, request)
	case dbsqlc.TenantMajorEngineVersionV1:
		return t.workerGetV1(ctx, tenant, request)
	default:
		return nil, fmt.Errorf("unsupported tenant version: %s", string(tenant.Version))
	}
}

func (t *WorkerService) workerGetV0(ctx echo.Context, tenant *dbsqlc.Tenant, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	worker := ctx.Get("worker").(*dbsqlc.GetWorkerByIdRow)

	slotState, recent, err := t.config.APIRepository.Worker().ListWorkerState(
		sqlchelpers.UUIDToStr(worker.Worker.TenantId),
		sqlchelpers.UUIDToStr(worker.Worker.ID),
		int(worker.Worker.MaxRuns),
	)

	if err != nil {
		return nil, err
	}

	workerIdToActionIds, err := t.config.APIRepository.Worker().GetWorkerActionsByWorkerId(
		sqlchelpers.UUIDToStr(worker.Worker.TenantId),
		[]string{sqlchelpers.UUIDToStr(worker.Worker.ID)},
	)

	if err != nil {
		return nil, err
	}

	actions, ok := workerIdToActionIds[sqlchelpers.UUIDToStr(worker.Worker.ID)]

	if !ok {
		return nil, fmt.Errorf("worker %s has no actions", sqlchelpers.UUIDToStr(worker.Worker.ID))
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

func (t *WorkerService) workerGetV1(ctx echo.Context, tenant *dbsqlc.Tenant, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	workerV0 := ctx.Get("worker").(*dbsqlc.GetWorkerByIdRow)

	worker, err := t.config.V1.Workers().GetWorkerById(sqlchelpers.UUIDToStr(workerV0.Worker.ID))

	if err != nil {
		return nil, err
	}

	slotState, recent, err := t.config.V1.Workers().ListWorkerState(
		sqlchelpers.UUIDToStr(worker.Worker.TenantId),
		sqlchelpers.UUIDToStr(worker.Worker.ID),
		int(worker.Worker.MaxRuns),
	)

	if err != nil {
		return nil, err
	}

	workerIdToActions, err := t.config.APIRepository.Worker().GetWorkerActionsByWorkerId(
		sqlchelpers.UUIDToStr(worker.Worker.TenantId),
		[]string{sqlchelpers.UUIDToStr(worker.Worker.ID)},
	)

	if err != nil {
		return nil, err
	}

	actions, ok := workerIdToActions[sqlchelpers.UUIDToStr(worker.Worker.ID)]
	if !ok {
		return nil, fmt.Errorf("worker %s has no actions", sqlchelpers.UUIDToStr(worker.Worker.ID))
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

	workerResp := *transformersv1.ToWorkerSqlc(&worker.Worker, &slots, &worker.WebhookUrl.String, actions)

	workerResp.RecentStepRuns = &respStepRuns
	workerResp.Slots = transformersv1.ToSlotState(slotState, slots)

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
