package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowService) WorkflowRunList(ctx echo.Context, request gen.WorkflowRunListRequestObject) (gen.WorkflowRunListResponseObject, error) {
	return gen.WorkflowRunList400JSONResponse(apierrors.NewAPIErrors(
		"WorkflowRunList is deprecated; please use V1WorkflowRunList instead",
	)), nil
}
