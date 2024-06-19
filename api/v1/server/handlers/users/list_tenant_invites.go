package users

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *UserService) UserListTenantInvites(ctx echo.Context, request gen.UserListTenantInvitesRequestObject) (gen.UserListTenantInvitesResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)

	invites, err := t.config.APIRepository.TenantInvite().ListTenantInvitesByEmail(user.Email)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.TenantInvite, len(invites))

	for i := range invites {
		rows[i] = *transformers.ToTenantInviteLink(&invites[i])
	}

	return gen.UserListTenantInvites200JSONResponse(gen.TenantInviteList200JSONResponse{
		Rows: &rows,
	}), nil

}
