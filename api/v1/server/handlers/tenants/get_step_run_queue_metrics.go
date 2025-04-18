package tenants

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantGetStepRunQueueMetrics(ctx echo.Context, request gen.TenantGetStepRunQueueMetricsRequestObject) (gen.TenantGetStepRunQueueMetricsResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return t.tenantGetStepRunQueueMetricsV0(ctx, tenant, request)
	case dbsqlc.TenantMajorEngineVersionV1:
		return t.tenantGetStepRunQueueMetricsV1(ctx, tenant, request)
	default:
		return nil, fmt.Errorf("unsupported tenant version: %s", string(tenant.Version))
	}
}

func (t *TenantService) tenantGetStepRunQueueMetricsV0(ctx echo.Context, tenant *dbsqlc.Tenant, request gen.TenantGetStepRunQueueMetricsRequestObject) (gen.TenantGetStepRunQueueMetricsResponseObject, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	stepRunQueueCounts, err := t.config.EngineRepository.StepRun().GetQueueCounts(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	queueCountsInterface := make(map[string]interface{})

	for k, v := range stepRunQueueCounts {
		queueCountsInterface[k] = v
	}

	resp := gen.TenantStepRunQueueMetrics{
		Queues: &queueCountsInterface,
	}

	return gen.TenantGetStepRunQueueMetrics200JSONResponse(resp), nil
}

func (t *TenantService) tenantGetStepRunQueueMetricsV1(ctx echo.Context, tenant *dbsqlc.Tenant, request gen.TenantGetStepRunQueueMetricsRequestObject) (gen.TenantGetStepRunQueueMetricsResponseObject, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	stepRunQueueCounts, err := t.config.V1.Tasks().GetQueueCounts(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	resp := gen.TenantStepRunQueueMetrics{
		Queues: &stepRunQueueCounts,
	}

	return gen.TenantGetStepRunQueueMetrics200JSONResponse(resp), nil
}
