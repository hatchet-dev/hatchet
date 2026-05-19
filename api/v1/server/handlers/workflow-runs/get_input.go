package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowRunsService) WorkflowRunGetInput(ctx echo.Context, request gen.WorkflowRunGetInputRequestObject) (gen.WorkflowRunGetInputResponseObject, error) {
	return gen.WorkflowRunGetInput400JSONResponse(apierrors.NewAPIErrors(
		"WorkflowRunGetInput is deprecated",
	)), nil
}
