package users

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
)

func (u *UserService) UserCreate(ctx echo.Context, request gen.UserCreateRequestObject) (gen.UserCreateResponseObject, error) {
	// check that the server supports local registration
	if !u.config.Auth.ConfigFile.BasicAuthEnabled {
		return gen.UserCreate405JSONResponse(
			apierrors.NewAPIErrors("local registration is not enabled"),
		), nil
	}

	if !u.config.Runtime.AllowSignup {
		return gen.UserCreate400JSONResponse(
			apierrors.NewAPIErrors("user signups are disabled"),
		), nil
	}

	// validate the request
	if apiErrors, err := u.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.UserCreate400JSONResponse(*apiErrors), nil
	}

	// check restricted email group
	if err := u.checkUserRestrictionsForEmail(u.config, string(request.Body.Email)); err != nil {
		return nil, err
	}

	// determine if the user exists before attempting to write the user
	_, err := u.config.APIRepository.User().GetUserByEmail(ctx.Request().Context(), string(request.Body.Email))

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		u.config.Logger.Err(err).Msg("failed to get user by email")
		return gen.UserCreate400JSONResponse(
			apierrors.NewAPIErrors(ErrRegistrationFailed),
		), nil
	}

	if err == nil {
		// user already exists, return consistent error
		return gen.UserCreate400JSONResponse(
			apierrors.NewAPIErrors(ErrRegistrationFailed),
		), nil
	}

	hashedPw, err := repository.HashPassword(request.Body.Password)

	if err != nil {
		u.config.Logger.Err(err).Msg("failed to hash password")
		return gen.UserCreate400JSONResponse(
			apierrors.NewAPIErrors(ErrRegistrationFailed),
		), nil
	}

	if hashedPw == nil {
		u.config.Logger.Error().Msg("hashed password is nil")
		return gen.UserCreate400JSONResponse(
			apierrors.NewAPIErrors(ErrRegistrationFailed),
		), nil
	}

	createOpts := &repository.CreateUserOpts{
		Email:         string(request.Body.Email),
		EmailVerified: repository.BoolPtr(u.config.Auth.ConfigFile.SetEmailVerified),
		Name:          repository.StringPtr(request.Body.Name),
		Password:      hashedPw,
	}

	// write the user to the db
	user, err := u.config.APIRepository.User().CreateUser(ctx.Request().Context(), createOpts)
	if err != nil {
		u.config.Logger.Err(err).Msg("failed to create user")
		return gen.UserCreate400JSONResponse(
			apierrors.NewAPIErrors(ErrRegistrationFailed),
		), nil
	}

	err = authn.NewSessionHelpers(u.config).SaveAuthenticated(ctx, user)

	if err != nil {
		u.config.Logger.Err(err).Msg("failed to save authenticated session")
		return gen.UserCreate400JSONResponse(
			apierrors.NewAPIErrors(ErrRegistrationFailed),
		), nil
	}

	u.config.Analytics.Enqueue(
		"user:create",
		sqlchelpers.UUIDToStr(user.ID),
		nil,
		map[string]interface{}{
			"email": request.Body.Email,
			"name":  request.Body.Name,
		},
	)

	return gen.UserCreate200JSONResponse(
		*transformers.ToUser(user, false, nil),
	), nil
}
