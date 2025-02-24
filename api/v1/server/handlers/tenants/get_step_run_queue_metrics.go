package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantGetStepRunQueueMetrics(ctx echo.Context, request gen.TenantGetStepRunQueueMetricsRequestObject) (gen.TenantGetStepRunQueueMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	stepRunQueueCounts, err := t.config.EngineRepository.StepRun().GetQueueCounts(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	resp := gen.TenantStepRunQueueMetrics{
		Queues: &stepRunQueueCounts,
	}

	return gen.TenantGetStepRunQueueMetrics200JSONResponse(resp), nil
}
