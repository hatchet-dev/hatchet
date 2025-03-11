package users

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (t *UserService) UserListTenantInvites(ctx echo.Context, request gen.UserListTenantInvitesRequestObject) (gen.UserListTenantInvitesResponseObject, error) {
	user := ctx.Get("user").(*dbsqlc.User)

	invites, err := t.config.APIRepository.TenantInvite().ListTenantInvitesByEmail(ctx.Request().Context(), user.Email)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.TenantInvite, len(invites))

	for i := range invites {
		rows[i] = *transformers.ToUserTenantInviteLink(invites[i])
	}

	return gen.UserListTenantInvites200JSONResponse(gen.TenantInviteList200JSONResponse{
		Rows: &rows,
	}), nil

}
