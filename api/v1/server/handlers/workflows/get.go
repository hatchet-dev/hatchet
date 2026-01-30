package workflows

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowGet(ctx echo.Context, request gen.WorkflowGetRequestObject) (gen.WorkflowGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	workflow := ctx.Get("workflow").(*sqlcv1.GetWorkflowByIdRow)

	if workflow == nil || workflow.WorkflowVersionId == uuid.Nil {
		return gen.WorkflowGet404JSONResponse(gen.APIErrors{}), nil
	}

	version, _, _, _, _, err := t.config.V1.Workflows().GetWorkflowVersionWithTriggers(ctx.Request().Context(), tenantId, sqlchelpers.UUIDToStr(workflow.WorkflowVersionId))

	if err != nil {
		return nil, err
	}

	resp := transformers.ToWorkflow(&workflow.Workflow, &version.WorkflowVersion)

	return gen.WorkflowGet200JSONResponse(*resp), nil
}
