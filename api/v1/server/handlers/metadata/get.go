package metadata

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/authmode"
)

func (u *MetadataService) MetadataGet(ctx echo.Context, request gen.MetadataGetRequestObject) (gen.MetadataGetResponseObject, error) {
	authTypes := []string{}

	if u.config.Auth.ConfigFile.BasicAuthEnabled {
		authTypes = append(authTypes, "basic")
	}

	if u.config.Auth.ConfigFile.Google.Enabled {
		authTypes = append(authTypes, "google")
	}

	if u.config.Auth.ConfigFile.Github.Enabled {
		authTypes = append(authTypes, "github")
	}

	if u.config.Auth.ConfigFile.OIDC.Enabled {
		authTypes = append(authTypes, "oidc")
	}

	pylonAppID := u.config.Pylon.AppID

	var posthogConfig *gen.APIMetaPosthog

	if u.config.FePosthog != nil {
		posthogConfig = &gen.APIMetaPosthog{
			ApiKey:  &u.config.FePosthog.ApiKey,
			ApiHost: &u.config.FePosthog.ApiHost,
		}
	}

	observabilityEnabled := u.config.Observability.Enabled

	prometheusServerEnabled := u.config.Prometheus.PrometheusServerURL != ""

	authDisabled := authmode.IsDisabled
	allowCreateTenant := u.config.Runtime.AllowCreateTenant
	if allowCreateTenant && u.config.Runtime.CreateTenantRequiresGlobalAdmin {
		allowCreateTenant = false
		userID, err := authn.NewSessionHelpers(u.config.SessionStore).GetKeyUuid(ctx, "user_id")
		if err == nil {
			allowCreateTenant, err = authn.IsCurrentOIDCGlobalAdmin(ctx.Request().Context(), u.config, *userID)
			if err != nil {
				u.config.Logger.Warn().Err(err).Msg("could not verify current OIDC global administrator access")
			}
		}
	}

	meta := gen.APIMeta{
		Auth: &gen.APIMetaAuth{
			Schemes: &authTypes,
		},
		PylonAppId:              &pylonAppID,
		Posthog:                 posthogConfig,
		AllowSignup:             &u.config.Runtime.AllowSignup,
		AllowInvites:            &u.config.Runtime.AllowInvites,
		AllowCreateTenant:       &allowCreateTenant,
		AllowChangePassword:     &u.config.Runtime.AllowChangePassword,
		ObservabilityEnabled:    &observabilityEnabled,
		PrometheusServerEnabled: &prometheusServerEnabled,
		AuthDisabled:            &authDisabled,
		Embedded:                &u.config.Runtime.Embedded,
	}

	if authDisabled {
		if embeddedToken := authmode.EmbeddedToken(); embeddedToken != "" {
			meta.AuthDisabledToken = &embeddedToken
		}
	}

	return gen.MetadataGet200JSONResponse(meta), nil
}
