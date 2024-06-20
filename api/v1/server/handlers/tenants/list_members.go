package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *TenantService) TenantMemberList(ctx echo.Context, request gen.TenantMemberListRequestObject) (gen.TenantMemberListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	members, err := t.config.APIRepository.Tenant().ListTenantMembers(tenant.ID)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.TenantMember, len(members))

	for i := range members {
		rows[i] = *transformers.ToTenantMember(&members[i])
	}

	return gen.TenantMemberList200JSONResponse{
		Rows: &rows,
	}, nil
}
