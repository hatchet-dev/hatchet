package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowScheduledTrigger(ctx echo.Context, request gen.WorkflowScheduledTriggerRequestObject) (gen.WorkflowScheduledTriggerResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	scheduled := ctx.Get("scheduled-workflow-run").(*sqlcv1.ListScheduledWorkflowsRow)

	if scheduled == nil {
		return gen.WorkflowScheduledTrigger404JSONResponse(apierrors.NewAPIErrors("Scheduled workflow not found.")), nil
	}

	_, err := t.proxyTriggerScheduledWorkflow.Do(ctx.Request().Context(), tenant, &contracts.TriggerScheduledWorkflowRunRequest{
		ScheduleId: scheduled.ID.String(),
	})

	if err != nil {
		return gen.WorkflowScheduledTrigger400JSONResponse(apierrors.NewAPIErrors("Failed to trigger scheduled workflow.")), nil
	}

	return gen.WorkflowScheduledTrigger200JSONResponse(
		*transformers.ToScheduledWorkflowsFromSQLC(scheduled),
	), nil
}
