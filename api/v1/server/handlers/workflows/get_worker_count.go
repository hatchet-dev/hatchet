package workflows

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) WorkflowGetWorkersCount(ctx echo.Context, request gen.WorkflowGetWorkersCountRequestObject) (gen.WorkflowGetWorkersCountResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	w := ctx.Get("workflow").(*dbsqlc.GetWorkflowByIdRow)
	workflow := sqlchelpers.UUIDToStr(w.Workflow.ID)

	freeSlotCount, maxSlotCount, err := t.config.APIRepository.Workflow().GetWorkflowWorkerCount(tenantId, workflow)

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
		WorkflowRunId: &workflow,
	}), nil

}
