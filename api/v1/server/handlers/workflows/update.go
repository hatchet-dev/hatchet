package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *WorkflowService) WorkflowUpdate(ctx echo.Context, request gen.WorkflowUpdateRequestObject) (gen.WorkflowUpdateResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	workflow := ctx.Get("workflow").(*dbsqlc.GetWorkflowByIdRow)

	opts := repository.UpdateWorkflowOpts{
		IsPaused: request.Body.IsPaused,
	}

	updated, err := t.config.APIRepository.Workflow().UpdateWorkflow(ctx.Request().Context(), tenant.ID, sqlchelpers.UUIDToStr(workflow.Workflow.ID), &opts)

	if err != nil {
		return nil, err
	}

	resp := transformers.ToWorkflowFromSQLC(updated)

	return gen.WorkflowUpdate200JSONResponse(*resp), nil
}
