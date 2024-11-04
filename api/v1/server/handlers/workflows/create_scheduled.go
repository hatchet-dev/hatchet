package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *WorkflowService) ScheduledWorkflowRunCreate(ctx echo.Context, request gen.ScheduledWorkflowRunCreateRequestObject) (gen.ScheduledWorkflowRunCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	scheduled, err := t.config.APIRepository.Workflow().CreateScheduledWorkflow(ctx.Request().Context(), tenant.ID, &repository.CreateScheduledWorkflowRunForWorkflowOpts{
		ScheduledTrigger:   request.Body.TriggerAt,
		Input:              request.Body.Input,
		AdditionalMetadata: request.Body.AdditionalMetadata,
		WorkflowId:         request.Workflow.String(),
	})

	if err != nil {
		return gen.ScheduledWorkflowRunCreate400JSONResponse(
			apierrors.NewAPIErrors(err.Error()),
		), nil
	}

	return gen.ScheduledWorkflowRunCreate200JSONResponse(
		*transformers.ToScheduledWorkflowsFromSQLC(scheduled),
	), nil
}
