package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowDelete(ctx echo.Context, request gen.WorkflowDeleteRequestObject) (gen.WorkflowDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	workflow := ctx.Get("workflow").(*sqlcv1.GetWorkflowByIdRow)

	_, err := t.config.V1.Workflows().DeleteWorkflow(ctx.Request().Context(), tenantId, workflow.Workflow.ID.String())

	if err != nil {
		return nil, err
	}

	return gen.WorkflowDelete204Response{}, nil
}
