package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (t *WorkflowService) WorkflowDelete(ctx echo.Context, request gen.WorkflowDeleteRequestObject) (gen.WorkflowDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := tenant.ID.String()
	workflow := ctx.Get("workflow").(*dbsqlc.GetWorkflowByIdRow)

	_, err := t.config.APIRepository.Workflow().DeleteWorkflow(ctx.Request().Context(), tenantId, workflow.Workflow.ID.String())

	if err != nil {
		return nil, err
	}

	return gen.WorkflowDelete204Response{}, nil
}
