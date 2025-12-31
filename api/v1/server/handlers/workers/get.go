package workers

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	transformersv1 "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (t *WorkerService) WorkerGet(ctx echo.Context, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	return t.workerGetV1(ctx, tenant, request)
}

func (t *WorkerService) workerGetV1(ctx echo.Context, tenant *sqlcv1.Tenant, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	workerV0 := ctx.Get("worker").(*sqlcv1.GetWorkerByIdRow)

	worker, err := t.config.V1.Workers().GetWorkerById(sqlchelpers.UUIDToStr(workerV0.Worker.ID))

	if err != nil {
		return nil, err
	}

	slotState, err := t.config.V1.Workers().ListWorkerState(
		sqlchelpers.UUIDToStr(worker.Worker.TenantId),
		sqlchelpers.UUIDToStr(worker.Worker.ID),
		int(worker.Worker.MaxRuns),
	)

	if err != nil {
		return nil, err
	}

	workerIdToActions, err := t.config.V1.Workers().GetWorkerActionsByWorkerId(
		sqlchelpers.UUIDToStr(worker.Worker.TenantId),
		[]string{sqlchelpers.UUIDToStr(worker.Worker.ID)},
	)

	if err != nil {
		return nil, err
	}

	workerWorkflows, err := t.config.V1.Workers().GetWorkerWorkflowsByWorkerId(tenant.ID.String(), worker.Worker.ID.String())

	if err != nil {
		return nil, err
	}

	actions, ok := workerIdToActions[sqlchelpers.UUIDToStr(worker.Worker.ID)]
	if !ok {
		return nil, fmt.Errorf("worker %s has no actions", sqlchelpers.UUIDToStr(worker.Worker.ID))
	}

	respStepRuns := make([]gen.RecentStepRuns, 0)

	slots := int(worker.RemainingSlots)

	workerResp := *transformersv1.ToWorkerSqlc(&worker.Worker, &slots, &worker.WebhookUrl.String, actions, &workerWorkflows)

	workerResp.RecentStepRuns = &respStepRuns
	workerResp.Slots = transformersv1.ToSlotState(slotState, slots)

	affinity, err := t.config.V1.Workers().ListWorkerLabels(
		sqlchelpers.UUIDToStr(worker.Worker.TenantId),
		sqlchelpers.UUIDToStr(worker.Worker.ID),
	)

	if err != nil {
		return nil, err
	}

	workerResp.Labels = transformers.ToWorkerLabels(affinity)

	return gen.WorkerGet200JSONResponse(workerResp), nil
}
