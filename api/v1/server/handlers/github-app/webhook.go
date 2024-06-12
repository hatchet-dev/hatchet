package githubapp

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"

	githubsdk "github.com/google/go-github/v57/github"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (g *GithubAppService) GithubUpdateGlobalWebhook(ctx echo.Context, req gen.GithubUpdateGlobalWebhookRequestObject) (gen.GithubUpdateGlobalWebhookResponseObject, error) {
	ghApp, err := GetGithubAppConfig(g.config)

	if err != nil {
		return nil, err
	}

	// validate the payload using the github webhook signing secret
	payload, err := githubsdk.ValidatePayload(ctx.Request(), []byte(ghApp.GetWebhookSecret()))

	if err != nil {
		return nil, err
	}

	event, err := githubsdk.ParseWebHook(githubsdk.WebHookType(ctx.Request()), payload)

	if err != nil {
		return nil, err
	}

	switch e := event.(type) {
	case *githubsdk.InstallationRepositoriesEvent:
		if *e.Action == "added" {
			err = g.handleInstallationEvent(*e.Sender.ID, e.Installation)
		}
	case *githubsdk.InstallationEvent:
		if *e.Action == "created" || *e.Action == "added" {
			err = g.handleInstallationEvent(*e.Sender.ID, e.Installation)
		}

		if *e.Action == "deleted" {
			err = g.handleDeletionEvent(e.Installation)
		}
	}

	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (g *GithubAppService) handleInstallationEvent(senderID int64, i *githubsdk.Installation) error {
	// make sure the sender exists in the database
	gao, err := g.config.APIRepository.Github().ReadGithubAppOAuthByGithubUserID(int(senderID))

	if err != nil {
		return err
	}

	_, err = g.config.APIRepository.Github().ReadGithubAppInstallationByInstallationAndAccountID(int(*i.ID), int(*i.Account.ID))

	if err != nil && errors.Is(err, db.ErrNotFound) {
		// insert account/installation pair into database
		_, err := g.config.APIRepository.Github().CreateInstallation(gao.GithubUserID, &repository.CreateInstallationOpts{
			InstallationID:          int(*i.ID),
			AccountID:               int(*i.Account.ID),
			AccountName:             *i.Account.Login,
			AccountAvatarURL:        i.Account.AvatarURL,
			InstallationSettingsURL: i.HTMLURL,
		})

		if err != nil {
			return err
		}

		return nil
	}

	// associate the github user id with this installation in the database
	_, err = g.config.APIRepository.Github().AddGithubUserIdToInstallation(int(*i.ID), int(*i.Account.ID), gao.GithubUserID)

	return err
}

func (g *GithubAppService) handleDeletionEvent(i *githubsdk.Installation) error {
	_, err := g.config.APIRepository.Github().ReadGithubAppInstallationByInstallationAndAccountID(int(*i.ID), int(*i.Account.ID))

	if err != nil {
		return err
	}

	_, err = g.config.APIRepository.Github().DeleteInstallation(int(*i.ID), int(*i.Account.ID))

	if err != nil {
		return err
	}

	return nil
}
