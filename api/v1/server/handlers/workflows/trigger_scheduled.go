package workflows

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/services/ticker"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (w *WorkflowService) WorkflowScheduledTrigger(ctx echo.Context, request gen.WorkflowScheduledTriggerRequestObject) (gen.WorkflowScheduledTriggerResponseObject, error) {
	scheduled := ctx.Get("scheduled-workflow-run").(*sqlcv1.ListScheduledWorkflowsRow)

	if scheduled == nil {
		return gen.WorkflowScheduledTrigger404JSONResponse(apierrors.NewAPIErrors("Scheduled workflow not found.")), nil
	}

	if scheduled.WorkflowRunId != nil {
		return gen.WorkflowScheduledTrigger400JSONResponse(
			apierrors.NewAPIErrors("Scheduled run has already been triggered."),
		), nil
	}

	externalId, err := ticker.RunScheduledWorkflow(
		ctx.Request().Context(),
		w.config.Logger,
		w.config.MessageQueueV1,
		w.config.V1,
		scheduled.TenantId,
		repository.RunScheduledWorkflowV1Opts{
			ID:                 scheduled.ID,
			Input:              scheduled.Input,
			AdditionalMetadata: scheduled.AdditionalMetadata,
			Priority:           &scheduled.Priority,
			TriggerAt:          time.Now().UTC(),
			WorkflowName:       scheduled.Name,
		},
	)

	// external id can be nil if idempotency collision happens
	// note: this will be fixed soon with the new idempotency improvements
	if err != nil || externalId == nil {
		return gen.WorkflowScheduledTrigger500JSONResponse(apierrors.NewAPIErrors("Failed to trigger scheduled workflow.")), nil
	}

	w.config.Analytics.Enqueue(
		ctx.Request().Context(),
		analytics.WorkflowRun, analytics.Create,
		externalId.String(),
		nil,
	)

	return gen.WorkflowScheduledTrigger200JSONResponse(gen.TriggerRunResult{
		ExternalId: *externalId,
	}), nil
}
