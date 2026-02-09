package workflows

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowCronUpdate(ctx echo.Context, request gen.WorkflowCronUpdateRequestObject) (gen.WorkflowCronUpdateResponseObject, error) {
	_ = ctx.Get("tenant").(*sqlcv1.Tenant)
	cron := ctx.Get("cron-workflow").(*sqlcv1.ListCronWorkflowsRow)

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	err := t.config.V1.WorkflowSchedules().UpdateCronWorkflow(
		dbCtx,
		cron.TenantId,
		cron.CronId,
		&v1.UpdateCronOpts{
			Enabled: request.Body.Enabled,
		},
	)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowCronUpdate204Response{}, nil
}
