package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantGetStepRunQueueMetrics(ctx echo.Context, request gen.TenantGetStepRunQueueMetricsRequestObject) (gen.TenantGetStepRunQueueMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()

	stepRunQueueCounts, err := t.config.V1.Tasks().GetQueueCounts(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	resp := gen.TenantStepRunQueueMetrics{
		Queues: &stepRunQueueCounts,
	}

	return gen.TenantGetStepRunQueueMetrics200JSONResponse(resp), nil
}
