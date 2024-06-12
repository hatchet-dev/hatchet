package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *TenantService) TenantAlertingSettingsGet(ctx echo.Context, request gen.TenantAlertingSettingsGetRequestObject) (gen.TenantAlertingSettingsGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	tenantAlerting, err := t.config.APIRepository.TenantAlertingSettings().GetTenantAlertingSettings(tenant.ID)

	if err != nil {
		return nil, err
	}

	return gen.TenantAlertingSettingsGet200JSONResponse(
		*transformers.ToTenantAlertingSettings(tenantAlerting),
	), nil
}
