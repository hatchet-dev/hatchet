package workflows

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) WorkflowCronDelete(ctx echo.Context, request gen.WorkflowCronDeleteRequestObject) (gen.WorkflowCronDeleteResponseObject, error) {
	populator := populator.FromContext(ctx)

	_, err := populator.GetTenant()
	if err != nil {
		return nil, err
	}
	cron, err := populator.GetCronWorkflow()
	if err != nil {
		return nil, err
	}

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	err = t.config.APIRepository.Workflow().DeleteCronWorkflow(dbCtx,
		sqlchelpers.UUIDToStr(cron.TenantId),
		sqlchelpers.UUIDToStr(cron.CronId),
	)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowCronDelete204Response{}, nil
}
