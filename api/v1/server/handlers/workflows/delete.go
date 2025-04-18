package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) WorkflowDelete(ctx echo.Context, request gen.WorkflowDeleteRequestObject) (gen.WorkflowDeleteResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	workflow := ctx.Get("workflow").(*dbsqlc.GetWorkflowByIdRow)

	_, err = t.config.APIRepository.Workflow().DeleteWorkflow(ctx.Request().Context(), tenantId, sqlchelpers.UUIDToStr(workflow.Workflow.ID))

	if err != nil {
		return nil, err
	}

	return gen.WorkflowDelete204Response{}, nil
}
