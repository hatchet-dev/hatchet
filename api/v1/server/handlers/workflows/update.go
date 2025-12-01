package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (t *WorkflowService) WorkflowUpdate(ctx echo.Context, request gen.WorkflowUpdateRequestObject) (gen.WorkflowUpdateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := tenant.ID.String()
	workflow := ctx.Get("workflow").(*dbsqlc.GetWorkflowByIdRow)

	opts := repository.UpdateWorkflowOpts{
		IsPaused: request.Body.IsPaused,
	}

	updated, err := t.config.APIRepository.Workflow().UpdateWorkflow(ctx.Request().Context(), tenantId, workflow.Workflow.ID.String(), &opts)

	if err != nil {
		return nil, err
	}

	resp := transformers.ToWorkflowFromSQLC(updated)

	return gen.WorkflowUpdate200JSONResponse(*resp), nil
}
