package users

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
)

func (u *UserService) UserCreate(ctx echo.Context, request gen.UserCreateRequestObject) (gen.UserCreateResponseObject, error) {

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
	existingUser, err := u.config.APIRepository.User().GetUserByEmail(string(request.Body.Email))

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

	if hashedPw == nil {
		return nil, errors.New("hashed password is nil")
	}

	createOpts := &repository.CreateUserOpts{
		Email:         string(request.Body.Email),
		EmailVerified: repository.BoolPtr(u.config.Auth.ConfigFile.SetEmailVerified),
		Name:          repository.StringPtr(request.Body.Name),
		Password:      hashedPw,
	}

	// write the user to the db
	user, err := u.config.APIRepository.User().CreateUser(createOpts)
	if err != nil {
		return nil, err
	}

	err = authn.NewSessionHelpers(u.config).SaveAuthenticated(ctx, user)

	if err != nil {
		return nil, err
	}

	u.config.Analytics.Enqueue(
		"user:create",
		user.ID,
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
