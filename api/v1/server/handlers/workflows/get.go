package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) WorkflowGet(ctx echo.Context, request gen.WorkflowGetRequestObject) (gen.WorkflowGetResponseObject, error) {
	populator := populator.FromContext(ctx)

	tenant, err := populator.GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	workflow, err := populator.GetWorkflow()
	if err != nil {
		return nil, err
	}

	if workflow == nil || !workflow.WorkflowVersionId.Valid {
		return gen.WorkflowGet404JSONResponse(gen.APIErrors{}), nil
	}

	version, _, _, _, err := t.config.APIRepository.Workflow().GetWorkflowVersionById(tenantId, sqlchelpers.UUIDToStr(workflow.WorkflowVersionId))

	if err != nil {
		return nil, err
	}

	resp := transformers.ToWorkflow(&workflow.Workflow, &version.WorkflowVersion)

	return gen.WorkflowGet200JSONResponse(*resp), nil
}
