package metadata

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/integrations/vcs"
)

func (u *MetadataService) MetadataListIntegrations(ctx echo.Context, request gen.MetadataListIntegrationsRequestObject) (gen.MetadataListIntegrationsResponseObject, error) {
	integrations := []gen.APIMetaIntegration{}

	if provider, exists := u.config.VCSProviders[vcs.VCSRepositoryKindGithub]; exists && provider != nil {
		integrations = append(integrations, gen.APIMetaIntegration{
			Enabled: true,
			Name:    "github",
		})
	}

	if _, exists := u.config.AdditionalOAuthConfigs["slack"]; exists {
		integrations = append(integrations, gen.APIMetaIntegration{
			Enabled: true,
			Name:    "slack",
		})
	}

	if u.config.Email.IsValid() {
		integrations = append(integrations, gen.APIMetaIntegration{
			Enabled: true,
			Name:    "email",
		})
	}

	return gen.MetadataListIntegrations200JSONResponse(integrations), nil
}
