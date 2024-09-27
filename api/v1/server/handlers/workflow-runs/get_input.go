package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowRunsService) WorkflowRunGetInput(ctx echo.Context, request gen.WorkflowRunGetInputRequestObject) (gen.WorkflowRunGetInputResponseObject, error) {

	input, err := t.config.EngineRepository.WorkflowRun().GetWorkflowRunInputData(request.Tenant.String(), request.WorkflowRun.String())

	if err != nil {
		return nil, err
	}

	return gen.WorkflowRunGetInput200JSONResponse(
		input,
	), nil
}
