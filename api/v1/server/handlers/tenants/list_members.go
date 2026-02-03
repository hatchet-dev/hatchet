package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantMemberList(ctx echo.Context, request gen.TenantMemberListRequestObject) (gen.TenantMemberListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	members, err := t.config.V1.Tenant().ListTenantMembers(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.TenantMember, len(members))

	for i := range members {
		rows[i] = *transformers.ToTenantMember(members[i])
	}

	return gen.TenantMemberList200JSONResponse{
		Rows: &rows,
	}, nil
}
