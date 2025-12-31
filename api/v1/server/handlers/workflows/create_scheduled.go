package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (t *WorkflowService) ScheduledWorkflowRunCreate(ctx echo.Context, request gen.ScheduledWorkflowRunCreateRequestObject) (gen.ScheduledWorkflowRunCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	workflow, err := t.config.EngineRepository.Workflow().GetWorkflowByName(ctx.Request().Context(), tenantId, request.Workflow)

	if err != nil {
		return gen.ScheduledWorkflowRunCreate400JSONResponse(apierrors.NewAPIErrors("workflow not found")), nil
	}

	var priority int32 = 1

	if request.Body.Priority != nil {
		priority = *request.Body.Priority
	}

	scheduled, err := t.config.V1.WorkflowSchedules().CreateScheduledWorkflow(ctx.Request().Context(), tenantId, &v1.CreateScheduledWorkflowRunForWorkflowOpts{
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

	correlationIdInterface, ok := (request.Body.AdditionalMetadata)[string(constants.CorrelationIdKey)]
	if ok {
		correlationId, ok := correlationIdInterface.(string)
		if ok {
			ctx.Set(constants.CorrelationIdKey.String(), correlationId)
		}
	}

	ctx.Set(constants.ResourceIdKey.String(), scheduled.ID.String())
	ctx.Set(constants.ResourceTypeKey.String(), constants.ResourceTypeScheduledWorkflow.String())

	return gen.ScheduledWorkflowRunCreate200JSONResponse(
		*transformers.ToScheduledWorkflowsFromSQLC(scheduled),
	), nil
}
