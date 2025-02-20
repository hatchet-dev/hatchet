package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *TenantService) AlertEmailGroupList(ctx echo.Context, request gen.AlertEmailGroupListRequestObject) (gen.AlertEmailGroupListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	emailGroups, err := t.config.APIRepository.TenantAlertingSettings().ListTenantAlertGroups(tenant.ID)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.TenantAlertEmailGroup, len(emailGroups))

	for i := range emailGroups {
		rows[i] = *transformers.ToTenantAlertEmailGroup(&emailGroups[i])
	}

	return gen.AlertEmailGroupList200JSONResponse{
		Rows: &rows,
	}, nil
}
