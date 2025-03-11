package users

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func (u *UserService) UserUpdateLogout(ctx echo.Context, request gen.UserUpdateLogoutRequestObject) (gen.UserUpdateLogoutResponseObject, error) {
	user := ctx.Get("user").(*dbsqlc.User)

	if err := authn.NewSessionHelpers(u.config).SaveUnauthenticated(ctx); err != nil {
		return nil, err
	}

	return gen.UserUpdateLogout200JSONResponse(
		*transformers.ToUser(user, false, nil),
	), nil
}
