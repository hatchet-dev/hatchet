package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *TenantService) TenantGetQueueMetrics(ctx echo.Context, request gen.TenantGetQueueMetricsRequestObject) (gen.TenantGetQueueMetricsResponseObject, error) {
	return gen.TenantGetQueueMetrics400JSONResponse(apierrors.NewAPIErrors(
		"TenantGetQueueMetrics is deprecated",
	)), nil
}
