package workflows

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/services/ticker"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (w *WorkflowService) WorkflowScheduledTrigger(ctx echo.Context, request gen.WorkflowScheduledTriggerRequestObject) (gen.WorkflowScheduledTriggerResponseObject, error) {
	scheduled := ctx.Get("scheduled-workflow-run").(*sqlcv1.ListScheduledWorkflowsRow)

	if scheduled == nil {
		return gen.WorkflowScheduledTrigger404JSONResponse(apierrors.NewAPIErrors("Scheduled workflow not found.")), nil
	}

	err := ticker.RunScheduledWorkflow(
		ctx.Request().Context(),
		w.config.Logger,
		w.config.MessageQueueV1,
		w.config.V1,
		scheduled.TenantId,
		repository.RunScheduledWorkflowV1Opts{
			Id:                 scheduled.ID,
			Input:              scheduled.Input,
			AdditionalMetadata: scheduled.AdditionalMetadata,
			Priority:           &scheduled.Priority,
			TriggerAt:          time.Now().UTC(),
			WorkflowName:       scheduled.Name,
		},
	)

	if err != nil {
		return gen.WorkflowScheduledTrigger400JSONResponse(apierrors.NewAPIErrors("Failed to trigger scheduled workflow.")), nil
	}

	return gen.WorkflowScheduledTrigger200JSONResponse(
		*transformers.ToScheduledWorkflowsFromSQLC(scheduled),
	), nil
}
