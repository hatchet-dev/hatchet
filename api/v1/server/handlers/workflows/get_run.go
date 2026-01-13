package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowService) WorkflowRunGet(ctx echo.Context, request gen.WorkflowRunGetRequestObject) (gen.WorkflowRunGetResponseObject, error) {
	return gen.WorkflowRunGet400JSONResponse(apierrors.NewAPIErrors(
		"WorkflowRunGet is deprecated",
	)), nil
}
