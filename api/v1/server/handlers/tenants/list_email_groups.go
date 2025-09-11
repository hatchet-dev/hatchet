package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) AlertEmailGroupList(ctx echo.Context, request gen.AlertEmailGroupListRequestObject) (gen.AlertEmailGroupListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	emailGroups, err := t.config.APIRepository.TenantAlertingSettings().ListTenantAlertGroups(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.TenantAlertEmailGroup, len(emailGroups))

	for i := range emailGroups {
		rows[i] = *transformers.ToTenantAlertEmailGroup(emailGroups[i])
	}

	return gen.AlertEmailGroupList200JSONResponse{
		Rows: &rows,
	}, nil
}
