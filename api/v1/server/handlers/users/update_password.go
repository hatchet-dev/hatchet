package users

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (u *UserService) UserUpdatePassword(ctx echo.Context, request gen.UserUpdatePasswordRequestObject) (gen.UserUpdatePasswordResponseObject, error) {
	// determine if the user exists before attempting to write the user
	existingUser := ctx.Get("user").(*db.UserModel)

	// check that the server supports local registration
	if !u.config.Auth.ConfigFile.BasicAuthEnabled {
		return gen.UserUpdatePassword405JSONResponse(
			apierrors.NewAPIErrors("local registration is not enabled"),
		), nil
	}

	// validate the request
	if apiErrors, err := u.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.UserUpdatePassword400JSONResponse(*apiErrors), nil
	}

	userPass, err := u.config.APIRepository.User().GetUserPassword(existingUser.ID)

	if err != nil {
		return nil, fmt.Errorf("could not get user password: %w", err)
	}

	if verified, err := repository.VerifyPassword(userPass.Hash, request.Body.Password); !verified || err != nil {
		return gen.UserUpdatePassword400JSONResponse(apierrors.NewAPIErrors("invalid password", "password")), nil
	}

	// Update the user

	newPass, err := repository.HashPassword(request.Body.NewPassword)

	if err != nil {
		return nil, fmt.Errorf("could not hash user password: %w", err)
	}

	user, err := u.config.APIRepository.User().UpdateUser(existingUser.ID, &repository.UpdateUserOpts{
		Password: newPass,
	})

	if err != nil {
		return nil, fmt.Errorf("could not update user: %w", err)
	}

	return gen.UserUpdatePassword200JSONResponse(
		*transformers.ToUser(user, true, nil),
	), nil
}
