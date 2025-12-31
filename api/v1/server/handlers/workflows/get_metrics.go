package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowService) WorkflowGetMetrics(ctx echo.Context, request gen.WorkflowGetMetricsRequestObject) (gen.WorkflowGetMetricsResponseObject, error) {
	panic("deprecated")
}
