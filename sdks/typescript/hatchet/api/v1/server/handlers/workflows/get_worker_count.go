package workflows

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *WorkflowService) WorkflowGetWorkersCount(ctx echo.Context, request gen.WorkflowGetWorkersCountRequestObject) (gen.WorkflowGetWorkersCountResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	w := ctx.Get("workflow").(*dbsqlc.GetWorkflowByIdRow)
	workflow := sqlchelpers.UUIDToStr(w.Workflow.ID)

	freeSlotCount, maxSlotCount, err := t.config.APIRepository.Workflow().GetWorkflowWorkerCount(tenant.ID, workflow)

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
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
