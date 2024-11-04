package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *WorkflowService) ScheduledWorkflowRunCreate(ctx echo.Context, request gen.ScheduledWorkflowRunCreateRequestObject) (gen.ScheduledWorkflowRunCreateResponseObject, error) {

	// t.

	return gen.ScheduledWorkflowRunCreate200JSONResponse(
		gen.ScheduledWorkflows{},
	), nil
}
