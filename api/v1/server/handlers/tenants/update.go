package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *TenantService) TenantUpdate(ctx echo.Context, request gen.TenantUpdateRequestObject) (gen.TenantUpdateResponseObject, error) {
	return gen.TenantUpdate400JSONResponse(apierrors.NewAPIErrors("this is disabled for the demo")), nil
}
