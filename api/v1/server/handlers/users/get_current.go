package users

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (u *UserService) UserGetCurrent(ctx echo.Context, request gen.UserGetCurrentRequestObject) (gen.UserGetCurrentResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)

	var hasPass bool

	pass, err := u.config.APIRepository.User().GetUserPassword(user.ID)

	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}

	if pass != nil {
		hasPass = true
	}

	return gen.UserGetCurrent200JSONResponse(
		*transformers.ToUser(user, hasPass),
	), nil
}
