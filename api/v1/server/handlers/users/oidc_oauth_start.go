package users

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/redirect"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (u *UserService) UserUpdateOidcOauthStart(ctx echo.Context, _ gen.UserUpdateOidcOauthStartRequestObject) (gen.UserUpdateOidcOauthStartResponseObject, error) {
	if u.config.Auth.OIDCOAuthConfig == nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, nil, "OIDC authentication is not configured.")
	}

	state, err := authn.NewSessionHelpers(u.config.SessionStore).SaveOAuthState(ctx, "oidc")

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Could not get cookie. Please make sure cookies are enabled.")
	}

	url := u.config.Auth.OIDCOAuthConfig.AuthCodeURL(state)

	return gen.UserUpdateOidcOauthStart302Response{
		Headers: gen.UserUpdateOidcOauthStart302ResponseHeaders{
			Location: url,
		},
	}, nil
}
