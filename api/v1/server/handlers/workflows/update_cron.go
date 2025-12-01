package workflows

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (t *WorkflowService) WorkflowCronUpdate(ctx echo.Context, request gen.WorkflowCronUpdateRequestObject) (gen.WorkflowCronUpdateResponseObject, error) {
	_ = ctx.Get("tenant").(*dbsqlc.Tenant)
	cron := ctx.Get("cron-workflow").(*dbsqlc.ListCronWorkflowsRow)

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	err := t.config.APIRepository.Workflow().UpdateCronWorkflow(
		dbCtx,
		cron.TenantId.String(),
		cron.CronId.String(),
		&repository.UpdateCronOpts{
			Enabled: request.Body.Enabled,
		},
	)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowCronUpdate204Response{}, nil
}
