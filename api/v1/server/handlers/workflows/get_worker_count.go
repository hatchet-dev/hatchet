package workflows

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *WorkflowService) WorkflowGetWorkersCount(ctx echo.Context, request gen.WorkflowGetWorkersCountRequestObject) (gen.WorkflowGetWorkersCountResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	workflow := ctx.Get("workflow").(*db.WorkflowModel)

	freeCount, maxCount, err := t.config.APIRepository.Workflow().GetWorkflowWorkerCount(tenant.ID, workflow.ID)

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return gen.WorkflowGetWorkersCount400JSONResponse(
				apierrors.NewAPIErrors("workflow not found"),
			), nil
		}

		return nil, err
	}

	return gen.WorkflowGetWorkersCount200JSONResponse(gen.WorkflowWorkersCount{
		FreeCount:     &freeCount,
		MaxCount:      &maxCount,
		WorkflowRunId: &workflow.ID,
	}), nil

}
