package users

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (u *UserService) UserUpdateLogin(ctx echo.Context, request gen.UserUpdateLoginRequestObject) (gen.UserUpdateLoginResponseObject, error) {
	// check that the server supports local registration
	if !u.config.Auth.ConfigFile.BasicAuthEnabled {
		return gen.UserUpdateLogin405JSONResponse(
			apierrors.NewAPIErrors("local registration is not enabled"),
		), nil
	}

	// check rate limiting by IP
	clientIP := ctx.RealIP()
	if !u.rateLimiter.IsAllowed("user:update_login", clientIP) {
		errMsg := fmt.Sprintf("%s for user login, try again in %s", ErrAuthAPIRateLimit, u.rateLimiter.GetWindow())
		u.config.Logger.Warn().Str("ip", clientIP).Msg(errMsg)
		return gen.UserUpdateLogin422JSONResponse(
			apierrors.NewAPIErrors(errMsg),
		), nil
	}

	// validate the request
	if apiErrors, err := u.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.UserUpdateLogin400JSONResponse(*apiErrors), nil
	}

	if err := u.checkUserRestrictionsForEmail(u.config, string(request.Body.Email)); err != nil {
		u.config.Logger.Err(err).Msg("email not in restricted domain")
		return gen.UserUpdateLogin400JSONResponse(
			apierrors.NewAPIErrors(ErrInvalidCredentials),
		), nil
	}

	// determine if the user exists before attempting to write the user
	existingUser, err := u.config.APIRepository.User().GetUserByEmail(ctx.Request().Context(), string(request.Body.Email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.UserUpdateLogin400JSONResponse(apierrors.NewAPIErrors(ErrInvalidCredentials)), nil
		}

		u.config.Logger.Err(err).Msg("failed to get user by email")
		return gen.UserUpdateLogin400JSONResponse(apierrors.NewAPIErrors(ErrInvalidCredentials)), nil
	}

	userPass, err := u.config.APIRepository.User().GetUserPassword(ctx.Request().Context(), sqlchelpers.UUIDToStr(existingUser.ID))

	if err != nil {
		u.config.Logger.Err(err).Msg("failed to get user password")
		return gen.UserUpdateLogin400JSONResponse(apierrors.NewAPIErrors(ErrInvalidCredentials)), nil
	}

	if verified, err := repository.VerifyPassword(userPass.Hash, request.Body.Password); !verified || err != nil {
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
