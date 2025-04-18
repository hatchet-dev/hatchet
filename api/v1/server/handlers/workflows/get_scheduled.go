package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (t *WorkflowService) WorkflowScheduledGet(ctx echo.Context, request gen.WorkflowScheduledGetRequestObject) (gen.WorkflowScheduledGetResponseObject, error) {
	// First check if scheduled object exists in context
	scheduledObj := ctx.Get("scheduled")
	if scheduledObj == nil {
		return gen.WorkflowScheduledGet404JSONResponse(apierrors.NewAPIErrors("Scheduled workflow not found.")), nil
	}

	scheduled, ok := scheduledObj.(*dbsqlc.ListScheduledWorkflowsRow)
	if !ok {
		return nil, echo.NewHTTPError(500, "Invalid scheduled workflow type in context")
	}

	return gen.WorkflowScheduledGet200JSONResponse(
		*transformers.ToScheduledWorkflowsFromSQLC(scheduled),
	), nil
}
