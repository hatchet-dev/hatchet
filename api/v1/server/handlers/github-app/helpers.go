package githubapp

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/integrations/vcs"
	"github.com/hatchet-dev/hatchet/internal/integrations/vcs/github"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"

	githubsdk "github.com/google/go-github/v57/github"
)

func GetGithubProvider(config *server.ServerConfig) (res github.GithubVCSProvider, reqErr error) {
	vcsProvider, exists := config.VCSProviders[vcs.VCSRepositoryKindGithub]

	if !exists {
		return res, fmt.Errorf("No Github app set up on this Hatchet instance.")
	}

	res, err := github.ToGithubVCSProvider(vcsProvider)

	if err != nil {
		return res, fmt.Errorf("Github app is improperly set up on this Hatchet instance.")
	}

	return res, nil
}

func GetGithubAppConfig(config *server.ServerConfig) (*github.GithubAppConf, error) {
	githubFact, reqErr := GetGithubProvider(config)

	if reqErr != nil {
		return nil, reqErr
	}

	return githubFact.GetGithubAppConfig(), nil
}

// GetGithubAppClientFromRequest gets the github app installation id from the request and authenticates
// using it and the private key
func GetGithubAppClientFromRequest(ctx echo.Context, config *server.ServerConfig) (*githubsdk.Client, *gen.APIErrors) {
	user := ctx.Get("user").(*db.UserModel)
	gai := ctx.Get("gh-installation").(*db.GithubAppInstallationModel)

	if canAccess, err := config.APIRepository.Github().CanUserAccessInstallation(gai.ID, user.ID); err != nil || !canAccess {
		respErr := apierrors.NewAPIErrors("User does not have access to the installation")
		return nil, &respErr
	}

	githubFact, err := GetGithubProvider(config)

	if err != nil {
		config.Logger.Err(err).Msg("Error getting github provider")
		respErr := apierrors.NewAPIErrors("Internal error")
		return nil, &respErr
	}

	res, err := githubFact.GetGithubAppConfig().GetGithubClient(int64(gai.InstallationID))

	if err != nil {
		config.Logger.Err(err).Msg("Error getting github client")
		respErr := apierrors.NewAPIErrors("Internal error")
		return nil, &respErr
	}

	return res, nil
}
