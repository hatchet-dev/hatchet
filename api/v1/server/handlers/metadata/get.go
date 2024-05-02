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

	pylonAppID := u.config.Pylon.AppID

	meta := gen.APIMeta{
		Auth: &gen.APIMetaAuth{
			Schemes: &authTypes,
		},
		PylonAppId: &pylonAppID,
	}

	return gen.MetadataGet200JSONResponse(meta), nil
}
