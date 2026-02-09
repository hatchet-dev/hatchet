package workers

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	transformersv1 "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkerService) WorkerGet(ctx echo.Context, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	return t.workerGetV1(ctx, tenant, request)
}

func (t *WorkerService) workerGetV1(ctx echo.Context, tenant *sqlcv1.Tenant, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	reqCtx := ctx.Request().Context()
	workerV0 := ctx.Get("worker").(*sqlcv1.GetWorkerByIdRow)

	worker, err := t.config.V1.Workers().GetWorkerById(reqCtx, tenant.ID, workerV0.Worker.ID)

	if err != nil {
		return nil, err
	}

	workerIdToActions, err := t.config.V1.Workers().GetWorkerActionsByWorkerId(
		reqCtx,
		worker.Worker.TenantId,
		[]uuid.UUID{worker.Worker.ID},
	)

	if err != nil {
		return nil, err
	}

	workerSlotConfig, err := buildWorkerSlotConfig(ctx.Request().Context(), t.config.V1.Workers(), worker.Worker.TenantId, []uuid.UUID{worker.Worker.ID})
	if err != nil {
		return nil, err
	}

	workerWorkflows, err := t.config.V1.Workers().GetWorkerWorkflowsByWorkerId(reqCtx, tenant.ID, worker.Worker.ID)

	if err != nil {
		return nil, err
	}

	actions, ok := workerIdToActions[worker.Worker.ID.String()]
	if !ok {
		return nil, fmt.Errorf("worker %s has no actions", worker.Worker.ID.String())
	}

	respStepRuns := make([]gen.RecentStepRuns, 0)

	slotConfig := workerSlotConfig[worker.Worker.ID]

	workerResp := *transformersv1.ToWorkerSqlc(&worker.Worker, slotConfig, actions, &workerWorkflows)

	workerResp.RecentStepRuns = &respStepRuns

	affinity, err := t.config.V1.Workers().ListWorkerLabels(
		reqCtx,
		worker.Worker.TenantId,
		worker.Worker.ID,
	)

	if err != nil {
		return nil, err
	}

	workerResp.Labels = transformers.ToWorkerLabels(affinity)

	return gen.WorkerGet200JSONResponse(workerResp), nil
}
