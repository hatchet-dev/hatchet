package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *TenantService) TenantGetPrometheusMetrics(ctx echo.Context, request gen.TenantGetPrometheusMetricsRequestObject) (gen.TenantGetPrometheusMetricsResponseObject, error) {
	// tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	// tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	return gen.TenantGetPrometheusMetrics200TextResponse(""), nil
}
