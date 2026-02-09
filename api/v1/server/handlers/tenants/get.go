package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantGet(ctx echo.Context, request gen.TenantGetRequestObject) (gen.TenantGetResponseObject, error) {
	maybeTenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	tenant := transformers.ToTenant(maybeTenant)

	return gen.TenantGet200JSONResponse(
		*tenant,
	), nil
}
