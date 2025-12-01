package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (t *TenantService) AlertEmailGroupDelete(ctx echo.Context, request gen.AlertEmailGroupDeleteRequestObject) (gen.AlertEmailGroupDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := tenant.ID.String()
	emailGroup := ctx.Get("alert-email-group").(*dbsqlc.TenantAlertEmailGroup)

	// delete the invite
	err := t.config.APIRepository.TenantAlertingSettings().DeleteTenantAlertGroup(ctx.Request().Context(), tenantId, emailGroup.ID.String())

	if err != nil {
		return nil, err
	}

	return gen.AlertEmailGroupDelete204Response{}, nil
}
