package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
)

func (t *WorkflowService) WorkflowScheduledGet(ctx echo.Context, request gen.WorkflowScheduledGetRequestObject) (gen.WorkflowScheduledGetResponseObject, error) {
	// First check if scheduled object exists in context
	populator := populator.FromContext(ctx)

	scheduled, err := populator.GetScheduledWorkflow()
	if err != nil {
		return nil, err
	}

	return gen.WorkflowScheduledGet200JSONResponse(
		*transformers.ToScheduledWorkflowsFromSQLC(scheduled),
	), nil
}
