package workers

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	transformersv1 "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
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
		worker.Worker.TenantId.String(),
		worker.Worker.ID.String(),
		int(worker.Worker.MaxRuns),
	)

	if err != nil {
		return nil, err
	}

	workerIdToActionIds, err := t.config.APIRepository.Worker().GetWorkerActionsByWorkerId(
		worker.Worker.TenantId.String(),
		[]string{worker.Worker.ID.String()},
	)

	if err != nil {
		return nil, err
	}

	actions, ok := workerIdToActionIds[worker.Worker.ID.String()]

	if !ok {
		return nil, fmt.Errorf("worker %s has no actions", worker.Worker.ID.String())
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
		worker.Worker.TenantId.String(),
		worker.Worker.ID.String(),
	)

	if err != nil {
		return nil, err
	}

	workerResp.Labels = transformers.ToWorkerLabels(affinity)

	return gen.WorkerGet200JSONResponse(workerResp), nil
}

func (t *WorkerService) workerGetV1(ctx echo.Context, tenant *dbsqlc.Tenant, request gen.WorkerGetRequestObject) (gen.WorkerGetResponseObject, error) {
	workerV0 := ctx.Get("worker").(*dbsqlc.GetWorkerByIdRow)

	worker, err := t.config.V1.Workers().GetWorkerById(workerV0.Worker.ID.String())

	if err != nil {
		return nil, err
	}

	slotState, recent, err := t.config.V1.Workers().ListWorkerState(
		worker.Worker.TenantId.String(),
		worker.Worker.ID.String(),
		int(worker.Worker.MaxRuns),
	)

	if err != nil {
		return nil, err
	}

	workerIdToActions, err := t.config.APIRepository.Worker().GetWorkerActionsByWorkerId(
		worker.Worker.TenantId.String(),
		[]string{worker.Worker.ID.String()},
	)

	if err != nil {
		return nil, err
	}

	workerWorkflows, err := t.config.APIRepository.Worker().GetWorkerWorkflowsByWorkerId(tenant.ID.String(), worker.Worker.ID.String())

	if err != nil {
		return nil, err
	}

	actions, ok := workerIdToActions[worker.Worker.ID.String()]
	if !ok {
		return nil, fmt.Errorf("worker %s has no actions", worker.Worker.ID.String())
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

	workerResp := *transformersv1.ToWorkerSqlc(&worker.Worker, &slots, &worker.WebhookUrl.String, actions, &workerWorkflows)

	workerResp.RecentStepRuns = &respStepRuns
	workerResp.Slots = transformersv1.ToSlotState(slotState, slots)

	affinity, err := t.config.APIRepository.Worker().ListWorkerLabels(
		worker.Worker.TenantId.String(),
		worker.Worker.ID.String(),
	)

	if err != nil {
		return nil, err
	}

	workerResp.Labels = transformers.ToWorkerLabels(affinity)

	return gen.WorkerGet200JSONResponse(workerResp), nil
}
