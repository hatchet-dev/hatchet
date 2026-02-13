package users

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func (u *UserService) UserUpdateLogin(ctx echo.Context, request gen.UserUpdateLoginRequestObject) (gen.UserUpdateLoginResponseObject, error) {
	// check that the server supports local registration
	if !u.config.Auth.ConfigFile.BasicAuthEnabled {
		return gen.UserUpdateLogin405JSONResponse(
			apierrors.NewAPIErrors("local registration is not enabled"),
		), nil
	}

	// validate the request
	if apiErrors, err := u.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.UserUpdateLogin400JSONResponse(*apiErrors), nil
	}

	if err := u.config.Auth.CheckEmailRestrictions(string(request.Body.Email)); err != nil {
		u.config.Logger.Err(err).Msg("email not in restricted domain")
		return gen.UserUpdateLogin400JSONResponse(
			apierrors.NewAPIErrors(ErrInvalidCredentials),
		), nil
	}

	// determine if the user exists before attempting to write the user
	existingUser, err := u.config.V1.User().GetUserByEmail(ctx.Request().Context(), string(request.Body.Email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.UserUpdateLogin400JSONResponse(apierrors.NewAPIErrors(ErrInvalidCredentials)), nil
		}

		u.config.Logger.Err(err).Msg("failed to get user by email")
		return gen.UserUpdateLogin400JSONResponse(apierrors.NewAPIErrors(ErrInvalidCredentials)), nil
	}

	userPass, err := u.config.V1.User().GetUserPassword(ctx.Request().Context(), existingUser.ID)

	if err != nil {
		u.config.Logger.Err(err).Msg("failed to get user password")
		return gen.UserUpdateLogin400JSONResponse(apierrors.NewAPIErrors(ErrInvalidCredentials)), nil
	}

	if verified, err := v1.VerifyPassword(userPass.Hash, request.Body.Password); !verified || err != nil {
		return gen.UserUpdateLogin400JSONResponse(apierrors.NewAPIErrors(ErrInvalidCredentials)), nil
	}

	err = authn.NewSessionHelpers(u.config).SaveAuthenticated(ctx, existingUser)

	if err != nil {
		u.config.Logger.Err(err).Msg("failed to save authenticated session")
		return gen.UserUpdateLogin400JSONResponse(apierrors.NewAPIErrors(ErrInvalidCredentials)), nil
	}

	return gen.UserUpdateLogin200JSONResponse(
		*transformers.ToUser(existingUser, false, nil),
	), nil
}
