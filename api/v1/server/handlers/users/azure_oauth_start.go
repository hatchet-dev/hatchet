package users

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/redirect"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (u *UserService) UserUpdateAzureOauthStart(ctx echo.Context, _ gen.UserUpdateAzureOauthStartRequestObject) (gen.UserUpdateAzureOauthStartResponseObject, error) {
	if !u.config.Runtime.AllowSignup {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, nil, "User signup is disabled.")
	}

	state, err := authn.NewSessionHelpers(u.config.SessionStore).SaveOAuthState(ctx, "azure")

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Could not get cookie. Please make sure cookies are enabled.")
	}

	url := u.config.Auth.AzureOAuthConfig.AuthCodeURL(state)

	return gen.UserUpdateAzureOauthStart302Response{
		Headers: gen.UserUpdateAzureOauthStart302ResponseHeaders{
			Location: url,
		},
	}, nil
}
