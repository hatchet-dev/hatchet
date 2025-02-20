package workflows

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *WorkflowService) WorkflowCronDelete(ctx echo.Context, request gen.WorkflowCronDeleteRequestObject) (gen.WorkflowCronDeleteResponseObject, error) {
	_ = ctx.Get("tenant").(*db.TenantModel)
	cron := ctx.Get("cron-workflow").(*dbsqlc.ListCronWorkflowsRow)

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	err := t.config.APIRepository.Workflow().DeleteCronWorkflow(dbCtx,
		sqlchelpers.UUIDToStr(cron.TenantId),
		sqlchelpers.UUIDToStr(cron.CronId),
	)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowCronDelete204Response{}, nil
}
