package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) AlertEmailGroupDelete(ctx echo.Context, request gen.AlertEmailGroupDeleteRequestObject) (gen.AlertEmailGroupDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()
	emailGroup := ctx.Get("alert-email-group").(*sqlcv1.TenantAlertEmailGroup)

	// delete the invite
	err := t.config.V1.TenantAlertingSettings().DeleteTenantAlertGroup(ctx.Request().Context(), tenantId, emailGroup.ID.String())

	if err != nil {
		return nil, err
	}

	return gen.AlertEmailGroupDelete204Response{}, nil
}
