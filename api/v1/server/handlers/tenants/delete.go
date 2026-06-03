package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantDelete(ctx echo.Context, request gen.TenantDeleteRequestObject) (gen.TenantDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID


	err := t.config.V1.Tenant().DeleteTenant(ctx.Request().Context(), tenantId)
	if err != nil {
		return nil, err
	}

	return gen.TenantDelete204Response{}, nil
}
