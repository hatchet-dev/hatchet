package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowService) WorkflowRunCancel(ctx echo.Context, request gen.WorkflowRunCancelRequestObject) (gen.WorkflowRunCancelResponseObject, error) {
	return gen.WorkflowRunCancel400JSONResponse(apierrors.NewAPIErrors(
		"WorkflowRunCancel is deprecated; please use V1TaskCancel instead",
	)), nil
}
