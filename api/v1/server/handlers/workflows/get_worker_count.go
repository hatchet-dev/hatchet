package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowService) WorkflowGetWorkersCount(ctx echo.Context, request gen.WorkflowGetWorkersCountRequestObject) (gen.WorkflowGetWorkersCountResponseObject, error) {
	panic("deprecated")
}
