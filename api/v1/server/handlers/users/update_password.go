package users

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (u *UserService) UserUpdatePassword(ctx echo.Context, request gen.UserUpdatePasswordRequestObject) (gen.UserUpdatePasswordResponseObject, error) {
	// determine if the user exists before attempting to write the user
	existingUser := ctx.Get("user").(*sqlcv1.User)

	if !u.config.Runtime.AllowChangePassword {
		return gen.UserUpdatePassword405JSONResponse(
			apierrors.NewAPIErrors("password changes are disabled"),
		), nil
	}

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

	userId := existingUser.ID.String()

	userPass, err := u.config.V1.User().GetUserPassword(ctx.Request().Context(), userId)

	if err != nil {
		u.config.Logger.Err(err).Msg("failed to get user password")
		return gen.UserUpdatePassword400JSONResponse(apierrors.NewAPIErrors(ErrInvalidCredentials)), nil
	}

	if verified, err := v1.VerifyPassword(userPass.Hash, request.Body.Password); !verified || err != nil {
		return gen.UserUpdatePassword400JSONResponse(apierrors.NewAPIErrors(ErrInvalidCredentials)), nil
	}

	// Update the user

	newPass, err := v1.HashPassword(request.Body.NewPassword)

	if err != nil {
		u.config.Logger.Err(err).Msg("failed to hash new password")
		return gen.UserUpdatePassword400JSONResponse(apierrors.NewAPIErrors(ErrInvalidCredentials)), nil
	}

	user, err := u.config.V1.User().UpdateUser(ctx.Request().Context(), userId, &v1.UpdateUserOpts{
		Password: newPass,
	})

	if err != nil {
		u.config.Logger.Err(err).Msg("failed to update user password")
		return gen.UserUpdatePassword400JSONResponse(apierrors.NewAPIErrors(ErrInvalidCredentials)), nil
	}

	return gen.UserUpdatePassword200JSONResponse(
		*transformers.ToUser(user, true, nil),
	), nil
}
