package githubapp

import (
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/redirect"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (g *GithubAppService) UserUpdateGithubAppOauthStart(ctx echo.Context, _ gen.UserUpdateGithubAppOauthStartRequestObject) (gen.UserUpdateGithubAppOauthStartResponseObject, error) {
	ghApp, err := GetGithubAppConfig(g.config)

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, g.config.Logger, err, "Github app is misconfigured on this Hatchet instance.")
	}

	state, err := authn.NewSessionHelpers(g.config).SaveOAuthState(ctx, "github")

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, g.config.Logger, err, "Could not get cookie. Please make sure cookies are enabled.")
	}

	url := ghApp.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return gen.UserUpdateGithubAppOauthStart302Response{
		Headers: gen.UserUpdateGithubAppOauthStart302ResponseHeaders{
			Location: url,
		},
	}, nil
}
