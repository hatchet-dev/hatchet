package slackapp

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/slack-go/slack"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/redirect"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (g *SlackAppService) UserUpdateSlackOauthCallback(ctx echo.Context, _ gen.UserUpdateSlackOauthCallbackRequestObject) (gen.UserUpdateSlackOauthCallbackResponseObject, error) {
	oauth, ok := g.config.AdditionalOAuthConfigs["slack"]

	if !ok {
		return nil, redirect.GetRedirectWithError(ctx, g.config.Logger, nil, "Slack OAuth is not configured on this Hatchet instance.")
	}

	sh := authn.NewSessionHelpers(g.config)

	tenantId, err := sh.GetKey(ctx, "tenant")

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, g.config.Logger, err, "Could not link Slack account. Please try again and make sure cookies are enabled.")
	}

	defer func() {
		err := sh.RemoveKey(ctx, "tenant")

		if err != nil {
			g.config.Logger.Error().Msgf("Could not remove tenant key: %v", err)
		}
	}()

	isValid, _, err := sh.ValidateOAuthState(ctx, "slack")

	if err != nil || !isValid {
		return nil, redirect.GetRedirectWithError(ctx, g.config.Logger, err, "Could not link Slack account. Please try again and make sure cookies are enabled.")
	}

	resp, err := slack.GetOAuthV2ResponseContext(
		ctx.Request().Context(),
		&http.Client{},
		oauth.ClientID,
		oauth.ClientSecret,
		ctx.Request().URL.Query().Get("code"),
		oauth.RedirectURL,
	)

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, g.config.Logger, err, "Forbidden")
	}

	webhookURLEncrypted, err := g.config.Encryption.Encrypt([]byte(resp.IncomingWebhook.URL), "incoming_webhook_url")

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, g.config.Logger, err, "Could not link Slack account. An internal error occurred.")
	}

	_, err = g.config.APIRepository.Slack().UpsertSlackWebhook(
		tenantId,
		&repository.UpsertSlackWebhookOpts{
			TeamId:      resp.Team.ID,
			TeamName:    resp.Team.Name,
			ChannelId:   resp.IncomingWebhook.ChannelID,
			ChannelName: resp.IncomingWebhook.Channel,
			WebhookURL:  webhookURLEncrypted,
		},
	)

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, g.config.Logger, err, "Could not link Slack account. An internal error occurred.")
	}

	return gen.UserUpdateSlackOauthCallback302Response{
		Headers: gen.UserUpdateSlackOauthCallback302ResponseHeaders{
			Location: "/tenant-settings/alerting",
		},
	}, nil
}
