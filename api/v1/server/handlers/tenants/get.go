package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (t *TenantService) TenantGet(ctx echo.Context, request gen.TenantGetRequestObject) (gen.TenantGetResponseObject, error) {
	maybeTenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	if maybeTenant == nil {
		return gen.TenantGet404JSONResponse(
			apierrors.NewAPIErrors("Tenant not found"),
		), nil
	}

	tenant := transformers.ToTenant(maybeTenant)

	return gen.TenantGet200JSONResponse(
		*tenant,
	), nil
}
