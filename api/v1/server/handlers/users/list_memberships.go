package users

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *UserService) TenantMembershipsList(ctx echo.Context, request gen.TenantMembershipsListRequestObject) (gen.TenantMembershipsListResponseObject, error) {
	user := ctx.Get("user").(*dbsqlc.User)
	userId := sqlchelpers.UUIDToStr(user.ID)

	memberships, err := t.config.APIRepository.User().ListTenantMemberships(ctx.Request().Context(), userId)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.TenantMember, len(memberships))

	for i, membership := range memberships {
		rows[i] = *transformers.ToTenantMember(membership)
	}

	return gen.TenantMembershipsList200JSONResponse(
		gen.UserTenantMembershipsList{
			Rows: &rows,
		},
	), nil
}
