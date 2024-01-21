package users

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
)

func (u *UserService) UserCreate(ctx echo.Context, request gen.UserCreateRequestObject) (gen.UserCreateResponseObject, error) {
	// validate the request
	if apiErrors, err := u.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.UserCreate400JSONResponse(*apiErrors), nil
	}

	// determine if the user exists before attempting to write the user
	existingUser, err := u.config.Repository.User().GetUserByEmail(string(request.Body.Email))

	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, err
	}

	if existingUser != nil {
		// just return bad request
		return gen.UserCreate400JSONResponse(
			apierrors.NewAPIErrors("Email is already registered."),
		), nil
	}

	hashedPw, err := repository.HashPassword(request.Body.Password)

	if err != nil {
		return nil, err
	}

	createOpts := &repository.CreateUserOpts{
		Email:         string(request.Body.Email),
		EmailVerified: repository.BoolPtr(u.config.Auth.SetEmailVerified),
		Name:          repository.StringPtr(request.Body.Name),
		Password:      *hashedPw,
	}

	// write the user to the db
	user, err := u.config.Repository.User().CreateUser(createOpts)
	if err != nil {
		return nil, err
	}

	err = authn.NewSessionHelpers(u.config).SaveAuthenticated(ctx, user)

	if err != nil {
		return nil, err
	}

	return gen.UserCreate200JSONResponse(
		*transformers.ToUser(user),
	), nil
}
