package users

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/labstack/echo/v4"
)

func (u *UserService) UserGetCurrent(ctx echo.Context, request gen.UserGetCurrentRequestObject) (gen.UserGetCurrentResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)

	return gen.UserGetCurrent200JSONResponse(
		*transformers.ToUser(user),
	), nil
}
