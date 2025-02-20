package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *TenantService) AlertEmailGroupDelete(ctx echo.Context, request gen.AlertEmailGroupDeleteRequestObject) (gen.AlertEmailGroupDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	emailGroup := ctx.Get("alert-email-group").(*db.TenantAlertEmailGroupModel)

	// delete the invite
	err := t.config.APIRepository.TenantAlertingSettings().DeleteTenantAlertGroup(tenant.ID, emailGroup.ID)

	if err != nil {
		return nil, err
	}

	return gen.AlertEmailGroupDelete204Response{}, nil
}
