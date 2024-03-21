package users

import (
	"errors"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
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

	// determine if the user exists before attempting to write the user
	existingUser, err := u.config.APIRepository.User().GetUserByEmail(string(request.Body.Email))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return gen.UserUpdateLogin400JSONResponse(apierrors.NewAPIErrors("user not found")), nil
		}

		return nil, err
	}

	userPass, err := u.config.APIRepository.User().GetUserPassword(existingUser.ID)

	if err != nil {
		return nil, fmt.Errorf("could not get user password: %w", err)
	}

	if verified, err := repository.VerifyPassword(userPass.Hash, request.Body.Password); !verified || err != nil {
		return gen.UserUpdateLogin400JSONResponse(apierrors.NewAPIErrors("invalid password")), nil
	}

	err = authn.NewSessionHelpers(u.config).SaveAuthenticated(ctx, existingUser)

	if err != nil {
		return nil, err
	}

	return gen.UserUpdateLogin200JSONResponse(
		*transformers.ToUser(existingUser),
	), nil
}
