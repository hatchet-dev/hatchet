package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowService) WorkflowRunCreate(ctx echo.Context, request gen.WorkflowRunCreateRequestObject) (gen.WorkflowRunCreateResponseObject, error) {
	return gen.WorkflowRunCreate400JSONResponse(apierrors.NewAPIErrors(
		"WorkflowRunCreate is deprecated; please use V1WorkflowRunCreate instead",
	)), nil
}
