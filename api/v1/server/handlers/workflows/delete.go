package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *WorkflowService) WorkflowDelete(ctx echo.Context, request gen.WorkflowDeleteRequestObject) (gen.WorkflowDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	workflow := ctx.Get("workflow").(*dbsqlc.GetWorkflowByIdRow)

	_, err := t.config.APIRepository.Workflow().DeleteWorkflow(tenant.ID, sqlchelpers.UUIDToStr(workflow.Workflow.ID))

	if err != nil {
		return nil, err
	}

	return gen.WorkflowDelete204Response{}, nil
}
