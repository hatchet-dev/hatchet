package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantGetTaskStats(ctx echo.Context, request gen.TenantGetTaskStatsRequestObject) (gen.TenantGetTaskStatsResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	stats, err := t.config.V1.Tasks().GetTaskStats(ctx.Request().Context(), tenant.ID)
	if err != nil {
		return nil, err
	}

	transformedStats := transformers.ToTaskStats(stats)

	return gen.TenantGetTaskStats200JSONResponse(transformedStats), nil
}
