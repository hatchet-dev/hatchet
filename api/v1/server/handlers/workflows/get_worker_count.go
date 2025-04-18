package workflows

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) WorkflowGetWorkersCount(ctx echo.Context, request gen.WorkflowGetWorkersCountRequestObject) (gen.WorkflowGetWorkersCountResponseObject, error) {
	populator := populator.FromContext(ctx)

	tenant, err := populator.GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	workflow, err := populator.GetWorkflow()
	if err != nil {
		return nil, err
	}
	workflowId := sqlchelpers.UUIDToStr(workflow.Workflow.ID)

	freeSlotCount, maxSlotCount, err := t.config.APIRepository.Workflow().GetWorkflowWorkerCount(tenantId, workflowId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.WorkflowGetWorkersCount400JSONResponse(
				apierrors.NewAPIErrors("workflow not found"),
			), nil
		}

		return nil, err
	}

	return gen.WorkflowGetWorkersCount200JSONResponse(gen.WorkflowWorkersCount{
		FreeSlotCount: &freeSlotCount,
		MaxSlotCount:  &maxSlotCount,
		WorkflowRunId: &workflowId,
	}), nil

}
