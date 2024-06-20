package users

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *UserService) TenantMembershipsList(ctx echo.Context, request gen.TenantMembershipsListRequestObject) (gen.TenantMembershipsListResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)

	memberships, err := t.config.APIRepository.User().ListTenantMemberships(user.ID)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.TenantMember, len(memberships))

	for i, membership := range memberships {
		membershipCp := membership
		rows[i] = *transformers.ToTenantMember(&membershipCp)
	}

	return gen.TenantMembershipsList200JSONResponse(
		gen.UserTenantMembershipsList{
			Rows: &rows,
		},
	), nil
}
