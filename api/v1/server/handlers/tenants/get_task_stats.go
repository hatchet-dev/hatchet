package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (t *TenantService) TenantGetTaskStats(ctx echo.Context, request gen.TenantGetTaskStatsRequestObject) (gen.TenantGetTaskStatsResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	stats, err := t.config.V1.Tasks().GetTaskStats(ctx.Request().Context(), tenant.ID.String())
	if err != nil {
		return nil, err
	}

	return gen.TenantGetTaskStats200JSONResponse(stats), nil
}
