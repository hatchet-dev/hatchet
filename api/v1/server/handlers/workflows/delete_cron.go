package workflows

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowCronDelete(ctx echo.Context, request gen.WorkflowCronDeleteRequestObject) (gen.WorkflowCronDeleteResponseObject, error) {
	_ = ctx.Get("tenant").(*sqlcv1.Tenant)
	cron := ctx.Get("cron-workflow").(*sqlcv1.ListCronWorkflowsRow)

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	err := t.config.V1.WorkflowSchedules().DeleteCronWorkflow(dbCtx,
		cron.TenantId.String(),
		cron.CronId.String(),
	)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowCronDelete204Response{}, nil
}
