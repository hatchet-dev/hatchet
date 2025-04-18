package users

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
)

func (u *UserService) UserUpdateLogout(ctx echo.Context, request gen.UserUpdateLogoutRequestObject) (gen.UserUpdateLogoutResponseObject, error) {
	populator := populator.FromContext(ctx)

	user, err := populator.GetUser()
	if err != nil {
		return nil, err
	}

	if err := authn.NewSessionHelpers(u.config).SaveUnauthenticated(ctx); err != nil {
		return nil, err
	}

	return gen.UserUpdateLogout200JSONResponse(
		*transformers.ToUser(user, false, nil),
	), nil
}
