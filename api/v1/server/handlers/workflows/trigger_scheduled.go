package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowScheduledTrigger(ctx echo.Context, request gen.WorkflowScheduledTriggerRequestObject) (gen.WorkflowScheduledTriggerResponseObject, error) {
	scheduled := ctx.Get("scheduled-workflow-run").(*sqlcv1.ListScheduledWorkflowsRow)

	if scheduled == nil {
		return gen.WorkflowScheduledTrigger404JSONResponse(apierrors.NewAPIErrors("Scheduled workflow not found.")), nil
	}

	return gen.WorkflowScheduledTrigger200JSONResponse(
		*transformers.ToScheduledWorkflowsFromSQLC(scheduled),
	), nil
}
