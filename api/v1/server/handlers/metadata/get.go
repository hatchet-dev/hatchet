package metadata

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
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

	meta := gen.APIMeta{
		Auth: &gen.APIMetaAuth{
			Schemes: &authTypes,
		},
		PylonAppId:           &pylonAppID,
		Posthog:              posthogConfig,
		AllowSignup:          &u.config.Runtime.AllowSignup,
		AllowInvites:         &u.config.Runtime.AllowInvites,
		AllowCreateTenant:    &u.config.Runtime.AllowCreateTenant,
		AllowChangePassword:  &u.config.Runtime.AllowChangePassword,
		ObservabilityEnabled: &observabilityEnabled,
	}

	return gen.MetadataGet200JSONResponse(meta), nil
}
