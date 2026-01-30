package workflows

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowScheduledUpdate(ctx echo.Context, request gen.WorkflowScheduledUpdateRequestObject) (gen.WorkflowScheduledUpdateResponseObject, error) {
	scheduled := ctx.Get("scheduled-workflow-run").(*sqlcv1.ListScheduledWorkflowsRow)

	if request.Body == nil {
		return gen.WorkflowScheduledUpdate400JSONResponse(apierrors.NewAPIErrors("Request body is required.")), nil
	}

	// Only allow updating scheduled runs created via API.
	if scheduled.Method != sqlcv1.WorkflowTriggerScheduledRefMethodsAPI {
		return gen.WorkflowScheduledUpdate403JSONResponse(apierrors.NewAPIErrors("Cannot update scheduled run created via code definition.")), nil
	}

	// If a scheduled run has already been triggered, it can no longer be rescheduled.
	if scheduled.WorkflowRunId == nil || *scheduled.WorkflowRunId != uuid.Nil {
		return gen.WorkflowScheduledUpdate400JSONResponse(apierrors.NewAPIErrors("Scheduled run has already been triggered and cannot be rescheduled.")), nil
	}

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	err := t.config.V1.WorkflowSchedules().UpdateScheduledWorkflow(
		dbCtx,
		scheduled.TenantId.String(),
		request.ScheduledWorkflowRun.String(),
		request.Body.TriggerAt,
	)
	if err != nil {
		return nil, err
	}

	updated, err := t.config.V1.WorkflowSchedules().GetScheduledWorkflow(
		dbCtx,
		scheduled.TenantId.String(),
		request.ScheduledWorkflowRun.String(),
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return gen.WorkflowScheduledUpdate404JSONResponse(apierrors.NewAPIErrors("Scheduled workflow not found.")), nil
	}

	return gen.WorkflowScheduledUpdate200JSONResponse(*transformers.ToScheduledWorkflowsFromSQLC(updated)), nil
}
