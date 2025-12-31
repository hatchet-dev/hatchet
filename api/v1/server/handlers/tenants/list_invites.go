package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
)

func (t *TenantService) TenantInviteList(ctx echo.Context, request gen.TenantInviteListRequestObject) (gen.TenantInviteListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	tenantInvites, err := t.config.V1.TenantInvite().ListTenantInvitesByTenantId(ctx.Request().Context(), tenantId, &v1.ListTenantInvitesOpts{
		Expired: repository.BoolPtr(false),
		Status:  repository.StringPtr("PENDING"),
	})

	if err != nil {
		return nil, err
	}

	rows := make([]gen.TenantInvite, len(tenantInvites))

	for i := range tenantInvites {
		rows[i] = *transformers.ToTenantInviteLink(tenantInvites[i])
	}

	return gen.TenantInviteList200JSONResponse{
		Rows: &rows,
	}, nil
}
