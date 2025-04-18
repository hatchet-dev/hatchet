package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantMemberList(ctx echo.Context, request gen.TenantMemberListRequestObject) (gen.TenantMemberListResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	members, err := t.config.APIRepository.Tenant().ListTenantMembers(ctx.Request().Context(), tenantId)

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
