package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *WorkflowService) WorkflowDelete(ctx echo.Context, request gen.WorkflowDeleteRequestObject) (gen.WorkflowDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	workflow := ctx.Get("workflow").(*db.WorkflowModel)

	workflow, err := t.config.APIRepository.Workflow().DeleteWorkflow(tenant.ID, workflow.ID)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowDelete204Response{}, nil
}
