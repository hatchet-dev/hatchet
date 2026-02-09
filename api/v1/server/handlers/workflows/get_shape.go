package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowService) WorkflowRunGetShape(ctx echo.Context, request gen.WorkflowRunGetShapeRequestObject) (gen.WorkflowRunGetShapeResponseObject, error) {
	return gen.WorkflowRunGetShape400JSONResponse(apierrors.NewAPIErrors(
		"WorkflowRunGetShape is deprecated",
	)), nil
}
