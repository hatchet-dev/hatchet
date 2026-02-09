package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowService) WorkflowRunGetMetrics(ctx echo.Context, request gen.WorkflowRunGetMetricsRequestObject) (gen.WorkflowRunGetMetricsResponseObject, error) {
	return gen.WorkflowRunGetMetrics400JSONResponse(apierrors.NewAPIErrors(
		"WorkflowRunGetMetrics is deprecated",
	)), nil
}
