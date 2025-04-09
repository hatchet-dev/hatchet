package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) ScheduledWorkflowRunCreate(ctx echo.Context, request gen.ScheduledWorkflowRunCreateRequestObject) (gen.ScheduledWorkflowRunCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	workflow, err := t.config.EngineRepository.Workflow().GetWorkflowByName(ctx.Request().Context(), tenantId, request.Workflow)

	if err != nil {
		return gen.ScheduledWorkflowRunCreate400JSONResponse(apierrors.NewAPIErrors("workflow not found")), nil
	}

	var priority int32

	if request.Body.Priority == nil {
		priority = 1
	} else {
		priority = *request.Body.Priority
	}

	scheduled, err := t.config.APIRepository.Workflow().CreateScheduledWorkflow(ctx.Request().Context(), tenantId, &repository.CreateScheduledWorkflowRunForWorkflowOpts{
		ScheduledTrigger:   request.Body.TriggerAt,
		Input:              request.Body.Input,
		AdditionalMetadata: request.Body.AdditionalMetadata,
		WorkflowId:         sqlchelpers.UUIDToStr(workflow.ID),
		Priority:           &priority,
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
