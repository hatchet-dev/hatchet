package githubapp

import (
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (g *GithubAppService) UserUpdateGithubOauthStart(ctx echo.Context, _ gen.UserUpdateGithubOauthStartRequestObject) (gen.UserUpdateGithubOauthStartResponseObject, error) {
	ghApp, err := GetGithubAppConfig(g.config)

	if err != nil {
		return nil, authn.GetRedirectWithError(ctx, g.config.Logger, err, "Github app is misconfigured on this Hatchet instance.")
	}

	state, err := authn.NewSessionHelpers(g.config).SaveOAuthState(ctx, "github")

	if err != nil {
		return nil, authn.GetRedirectWithError(ctx, g.config.Logger, err, "Could not get cookie. Please make sure cookies are enabled.")
	}

	url := ghApp.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return gen.UserUpdateGithubOauthStart302Response{
		Headers: gen.UserUpdateGithubOauthStart302ResponseHeaders{
			Location: url,
		},
	}, nil
}
