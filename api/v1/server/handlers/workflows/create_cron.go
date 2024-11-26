package workflows

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *WorkflowService) CronWorkflowTriggerCreate(ctx echo.Context, request gen.CronWorkflowTriggerCreateRequestObject) (gen.CronWorkflowTriggerCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	if request.Body.CronName == "" {
		return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors("cron name is required")), nil
	}

	workflow, err := t.config.EngineRepository.Workflow().GetWorkflowByName(ctx.Request().Context(), tenant.ID, request.Workflow)

	if err != nil {
		return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors("workflow not found")), nil
	}

	cronTrigger, err := t.config.APIRepository.Workflow().CreateCronWorkflow(
		ctx.Request().Context(), tenant.ID, &repository.CreateCronWorkflowTriggerOpts{
			Name:               request.Body.CronName,
			Cron:               request.Body.CronExpression,
			Input:              request.Body.Input,
			AdditionalMetadata: request.Body.AdditionalMetadata,
			WorkflowId:         sqlchelpers.UUIDToStr(workflow.ID),
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
