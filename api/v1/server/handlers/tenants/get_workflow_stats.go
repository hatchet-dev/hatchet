package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantGetWorkflowStats(ctx echo.Context, request gen.TenantGetWorkflowStatsRequestObject) (gen.TenantGetWorkflowStatsResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	stats, err := t.config.V1.OLAP().GetWorkflowStats(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	return gen.TenantGetWorkflowStats200JSONResponse(stats), nil
}
