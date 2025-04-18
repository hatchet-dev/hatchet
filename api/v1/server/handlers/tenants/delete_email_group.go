package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) AlertEmailGroupDelete(ctx echo.Context, request gen.AlertEmailGroupDeleteRequestObject) (gen.AlertEmailGroupDeleteResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	emailGroup := ctx.Get("alert-email-group").(*dbsqlc.TenantAlertEmailGroup)

	// delete the invite
	err = t.config.APIRepository.TenantAlertingSettings().DeleteTenantAlertGroup(ctx.Request().Context(), tenantId, sqlchelpers.UUIDToStr(emailGroup.ID))

	if err != nil {
		return nil, err
	}

	return gen.AlertEmailGroupDelete204Response{}, nil
}
