package workflows

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *WorkflowService) WorkflowScheduledGet(ctx echo.Context, request gen.WorkflowScheduledGetRequestObject) (gen.WorkflowScheduledGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	scheduled, err := t.config.APIRepository.WorkflowRun().GetScheduledWorkflow(dbCtx, tenant.ID, request.ScheduledId.String())

	if err != nil {
		return nil, err
	}

	if scheduled == nil {
		return gen.WorkflowScheduledGet404JSONResponse(apierrors.NewAPIErrors("Scheduled workflow not found.")), nil
	}

	return gen.WorkflowScheduledGet200JSONResponse(
		*transformers.ToScheduledWorkflowsFromSQLC(scheduled),
	), nil
}
