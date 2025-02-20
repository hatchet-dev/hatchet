package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

func (t *WorkflowService) WorkflowScheduledGet(ctx echo.Context, request gen.WorkflowScheduledGetRequestObject) (gen.WorkflowScheduledGetResponseObject, error) {
	scheduled := ctx.Get("scheduled").(*dbsqlc.ListScheduledWorkflowsRow)

	if scheduled == nil {
		return gen.WorkflowScheduledGet404JSONResponse(apierrors.NewAPIErrors("Scheduled workflow not found.")), nil
	}

	return gen.WorkflowScheduledGet200JSONResponse(
		*transformers.ToScheduledWorkflowsFromSQLC(scheduled),
	), nil
}
