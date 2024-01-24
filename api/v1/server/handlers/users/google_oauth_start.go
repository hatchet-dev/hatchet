package users

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (u *UserService) UserUpdateOauthStart(ctx echo.Context, _ gen.UserUpdateOauthStartRequestObject) (gen.UserUpdateOauthStartResponseObject, error) {
	state, err := authn.NewSessionHelpers(u.config).SaveOAuthState(ctx)

	if err != nil {
		return nil, authn.GetRedirectWithError(ctx, u.config.Logger, err, "Could not get cookie. Please make sure cookies are enabled.")
	}

	url := u.config.Auth.GoogleOAuthConfig.AuthCodeURL(state)

	return gen.UserUpdateOauthStart302Response{
		Headers: gen.UserUpdateOauthStart302ResponseHeaders{
			Location: url,
		},
	}, nil
}
