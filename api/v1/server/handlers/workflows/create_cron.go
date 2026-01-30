package workflows

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) CronWorkflowTriggerCreate(ctx echo.Context, request gen.CronWorkflowTriggerCreateRequestObject) (gen.CronWorkflowTriggerCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()

	if request.Body.CronName == "" {
		return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors("cron name is required")), nil
	}

	workflow, err := t.config.V1.Workflows().GetWorkflowByName(ctx.Request().Context(), tenantId, request.Workflow)

	if err != nil {
		return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors("workflow not found")), nil
	}

	var priority int32 = 1

	if request.Body.Priority != nil {
		priority = *request.Body.Priority
	}

	inputBytes, err := json.Marshal(request.Body.Input)

	if err != nil {
		return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors("invalid input format")), nil
	}

	if err := v1.ValidateJSONB(inputBytes, "input"); err != nil {
		return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
	}

	additionalMetaBytes, err := json.Marshal(request.Body.AdditionalMetadata)

	if err != nil {
		return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors("invalid additional metadata format")), nil
	}

	if err := v1.ValidateJSONB(additionalMetaBytes, "additionalMetadata"); err != nil {
		return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
	}

	cronTrigger, err := t.config.V1.WorkflowSchedules().CreateCronWorkflow(
		ctx.Request().Context(), uuid.MustParse(tenantId), &v1.CreateCronWorkflowTriggerOpts{
			Name:               request.Body.CronName,
			Cron:               request.Body.CronExpression,
			Input:              request.Body.Input,
			AdditionalMetadata: request.Body.AdditionalMetadata,
			WorkflowId:         workflow.ID,
			Priority:           &priority,
		},
	)

	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors("cron trigger with that name-expression pair already exists")), nil
		}

		return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors("error creating cron trigger")), nil
	}

	return gen.CronWorkflowTriggerCreate200JSONResponse(
		*transformers.ToCronWorkflowsFromSQLC(cronTrigger),
	), nil
}
