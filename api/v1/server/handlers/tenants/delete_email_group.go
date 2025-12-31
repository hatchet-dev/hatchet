package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (t *TenantService) AlertEmailGroupDelete(ctx echo.Context, request gen.AlertEmailGroupDeleteRequestObject) (gen.AlertEmailGroupDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	emailGroup := ctx.Get("alert-email-group").(*sqlcv1.TenantAlertEmailGroup)

	// delete the invite
	err := t.config.V1.TenantAlertingSettings().DeleteTenantAlertGroup(ctx.Request().Context(), tenantId, sqlchelpers.UUIDToStr(emailGroup.ID))

	if err != nil {
		return nil, err
	}

	return gen.AlertEmailGroupDelete204Response{}, nil
}
