package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantEntitlementsGet(ctx echo.Context, request gen.TenantEntitlementsGetRequestObject) (gen.TenantEntitlementsGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	entitlements, err := t.config.V1.TenantEntitlement().GetEntitlements(ctx.Request().Context(), tenant.ID)

	if err != nil {
		return nil, err
	}

	return gen.TenantEntitlementsGet200JSONResponse{
		AuditLogs:                       entitlements.AuditLogs,
		PrometheusMetrics:               entitlements.PrometheusMetrics,
		StrictAdditionalMetadataFilters: entitlements.StrictAdditionalMetadataFilters,
		DagOperator:                     entitlements.DAGOperator,
		DurableStreams:                  entitlements.DurableStreams,
		ServerlessOperator:              entitlements.ServerlessOperator,
	}, nil
}
