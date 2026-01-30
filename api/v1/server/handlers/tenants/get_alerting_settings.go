package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantAlertingSettingsGet(ctx echo.Context, request gen.TenantAlertingSettingsGetRequestObject) (gen.TenantAlertingSettingsGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()

	tenantAlerting, err := t.config.V1.TenantAlertingSettings().GetTenantAlertingSettings(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	return gen.TenantAlertingSettingsGet200JSONResponse(
		*transformers.ToTenantAlertingSettings(tenantAlerting.Settings),
	), nil
}
