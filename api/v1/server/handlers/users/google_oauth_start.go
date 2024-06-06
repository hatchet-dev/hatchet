package users

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/redirect"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (u *UserService) UserUpdateGoogleOauthStart(ctx echo.Context, _ gen.UserUpdateGoogleOauthStartRequestObject) (gen.UserUpdateGoogleOauthStartResponseObject, error) {
	state, err := authn.NewSessionHelpers(u.config).SaveOAuthState(ctx, "google")

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Could not get cookie. Please make sure cookies are enabled.")
	}

	url := u.config.Auth.GoogleOAuthConfig.AuthCodeURL(state)

	return gen.UserUpdateGoogleOauthStart302Response{
		Headers: gen.UserUpdateGoogleOauthStart302ResponseHeaders{
			Location: url,
		},
	}, nil
}
