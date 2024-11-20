package workflows

import (
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/rand"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *WorkflowService) CronWorkflowTriggerCreate(ctx echo.Context, request gen.CronWorkflowTriggerCreateRequestObject) (gen.CronWorkflowTriggerCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	if request.Body.CronName == "" {
		// Generate a random string for the cron name if not provided
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		b := make([]byte, 8)
		for i := range b {
			b[i] = charset[rand.Intn(len(charset))]
		}
		request.Body.CronName = string(b)
	}

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
