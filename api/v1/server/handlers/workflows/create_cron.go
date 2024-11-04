package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *WorkflowService) CronWorkflowTriggerCreate(ctx echo.Context, request gen.CronWorkflowTriggerCreateRequestObject) (gen.CronWorkflowTriggerCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	cronTrigger, err := t.config.APIRepository.Workflow().CreateCronWorkflow(
		ctx.Request().Context(), tenant.ID, &repository.CreateCronWorkflowTriggerOpts{
			Name:               request.Body.CronName,
			Cron:               request.Body.CronExpression,
			Input:              request.Body.Input,
			AdditionalMetadata: request.Body.AdditionalMetadata,
			WorkflowId:         request.Workflow.String(),
		},
	)

	if err != nil {
		return gen.CronWorkflowTriggerCreate400JSONResponse(apierrors.NewAPIErrors(err.Error())), err
	}

	return gen.CronWorkflowTriggerCreate200JSONResponse(
		*transformers.ToCronWorkflowsFromSQLC(cronTrigger),
	), nil
}
