package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowService) WorkflowUpdate(ctx echo.Context, request gen.WorkflowUpdateRequestObject) (gen.WorkflowUpdateResponseObject, error) {
	panic("deprecated")
}
