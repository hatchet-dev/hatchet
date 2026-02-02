package workflows

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowCronGet(ctx echo.Context, request gen.WorkflowCronGetRequestObject) (gen.WorkflowCronGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	scheduled, err := t.config.V1.WorkflowSchedules().GetCronWorkflow(dbCtx, tenant.ID, request.CronWorkflow)

	if err != nil {
		return nil, err
	}

	if scheduled == nil {
		return gen.WorkflowCronGet404JSONResponse(apierrors.NewAPIErrors("Scheduled workflow not found.")), nil
	}

	return gen.WorkflowCronGet200JSONResponse(
		*transformers.ToCronWorkflowsFromSQLC(scheduled),
	), nil
}
