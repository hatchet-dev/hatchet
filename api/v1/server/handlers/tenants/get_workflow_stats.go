package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (t *TenantService) TenantGetWorkflowStats(ctx echo.Context, request gen.TenantGetWorkflowStatsRequestObject) (gen.TenantGetWorkflowStatsResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	stats, err := t.config.V1.Tasks().GetWorkflowStats(ctx.Request().Context(), tenant.ID.String())

	if err != nil {
		return nil, err
	}

	return gen.TenantGetWorkflowStats200JSONResponse(stats), nil
}
