package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *WorkflowService) WorkflowGet(ctx echo.Context, request gen.WorkflowGetRequestObject) (gen.WorkflowGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	workflow := ctx.Get("workflow").(*dbsqlc.GetWorkflowByIdRow)

	if workflow == nil || !workflow.WorkflowVersionId.Valid {
		return gen.WorkflowGet404JSONResponse(gen.APIErrors{}), nil
	}

	version, _, _, _, err := t.config.APIRepository.Workflow().GetWorkflowVersionById(tenant.ID, sqlchelpers.UUIDToStr(workflow.WorkflowVersionId))

	if err != nil {
		return nil, err
	}

	resp := transformers.ToWorkflow(&workflow.Workflow, &version.WorkflowVersion)

	return gen.WorkflowGet200JSONResponse(*resp), nil
}
