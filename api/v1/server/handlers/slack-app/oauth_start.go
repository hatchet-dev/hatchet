package slackapp

import (
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/redirect"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (g *SlackAppService) UserUpdateSlackOauthStart(ctx echo.Context, _ gen.UserUpdateSlackOauthStartRequestObject) (gen.UserUpdateSlackOauthStartResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	oauth, ok := g.config.AdditionalOAuthConfigs["slack"]

	if !ok {
		return nil, redirect.GetRedirectWithError(ctx, g.config.Logger, nil, "Slack OAuth is not configured on this Hatchet instance.")
	}

	sh := authn.NewSessionHelpers(g.config)

	if err := sh.SaveKV(ctx, "tenant", tenantId); err != nil {
		return nil, redirect.GetRedirectWithError(ctx, g.config.Logger, err, "Could not get cookie. Please make sure cookies are enabled.")
	}

	state, err := sh.SaveOAuthState(ctx, "slack")

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, g.config.Logger, err, "Could not get cookie. Please make sure cookies are enabled.")
	}

	url := oauth.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return gen.UserUpdateSlackOauthStart302Response{
		Headers: gen.UserUpdateSlackOauthStart302ResponseHeaders{
			Location: url,
		},
	}, nil
}
