package workflows

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) ScheduledWorkflowRunCreate(ctx echo.Context, request gen.ScheduledWorkflowRunCreateRequestObject) (gen.ScheduledWorkflowRunCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()

	workflow, err := t.config.V1.Workflows().GetWorkflowByName(ctx.Request().Context(), tenantId, request.Workflow)

	if err != nil {
		return gen.ScheduledWorkflowRunCreate400JSONResponse(apierrors.NewAPIErrors("workflow not found")), nil
	}

	var priority int32 = 1

	if request.Body.Priority != nil {
		priority = *request.Body.Priority
	}

	var input, additionalMetadata []byte

	if request.Body.Input != nil {
		input, err = json.Marshal(request.Body.Input)

		if err != nil {
			return gen.ScheduledWorkflowRunCreate400JSONResponse(
				apierrors.NewAPIErrors("could not marshal input"),
			), nil
		}
	}

	if request.Body.AdditionalMetadata != nil {
		additionalMetadata, err = json.Marshal(request.Body.AdditionalMetadata)

		if err != nil {
			return gen.ScheduledWorkflowRunCreate400JSONResponse(
				apierrors.NewAPIErrors("could not marshal additionalMetadata"),
			), nil
		}
	}

	scheduled, err := t.config.V1.WorkflowSchedules().CreateScheduledWorkflow(ctx.Request().Context(), uuid.MustParse(tenantId), &v1.CreateScheduledWorkflowRunForWorkflowOpts{
		ScheduledTrigger:   request.Body.TriggerAt,
		Input:              input,
		AdditionalMetadata: additionalMetadata,
		WorkflowId:         workflow.ID,
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
