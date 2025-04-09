package workflows

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) CronWorkflowTriggerCreate(ctx echo.Context, request gen.CronWorkflowTriggerCreateRequestObject) (gen.CronWorkflowTriggerCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	if request.Body.CronName == "" {
		return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors("cron name is required")), nil
	}

	workflow, err := t.config.EngineRepository.Workflow().GetWorkflowByName(ctx.Request().Context(), tenantId, request.Workflow)

	if err != nil {
		return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors("workflow not found")), nil
	}

	var priority int32

	if request.Body.Priority == nil {
		priority = 1
	} else {
		priority = *request.Body.Priority
	}

	cronTrigger, err := t.config.APIRepository.Workflow().CreateCronWorkflow(
		ctx.Request().Context(), tenantId, &repository.CreateCronWorkflowTriggerOpts{
			Name:               request.Body.CronName,
			Cron:               request.Body.CronExpression,
			Input:              request.Body.Input,
			AdditionalMetadata: request.Body.AdditionalMetadata,
			WorkflowId:         sqlchelpers.UUIDToStr(workflow.ID),
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
