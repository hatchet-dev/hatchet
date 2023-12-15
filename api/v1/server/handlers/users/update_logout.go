package users

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/labstack/echo/v4"
)

func (u *UserService) UserUpdateLogout(ctx echo.Context, request gen.UserUpdateLogoutRequestObject) (gen.UserUpdateLogoutResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)

	if err := authn.NewSessionHelpers(u.config).SaveUnauthenticated(ctx); err != nil {
		return nil, err
	}

	return gen.UserUpdateLogout200JSONResponse(
		*transformers.ToUser(user),
	), nil
}
