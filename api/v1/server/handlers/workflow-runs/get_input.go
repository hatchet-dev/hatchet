package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *WorkflowRunsService) WorkflowRunGetInput(ctx echo.Context, request gen.WorkflowRunGetInputRequestObject) (gen.WorkflowRunGetInputResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	run := ctx.Get("workflow-run").(*db.WorkflowRunModel)

	input, err := t.config.APIRepository.WorkflowRun().GetWorkflowRunInputData(tenant.ID, run.ID)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowRunGetInput200JSONResponse(
		input,
	), nil
}
