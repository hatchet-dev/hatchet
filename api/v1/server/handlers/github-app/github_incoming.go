package githubapp

import (
	"fmt"
	"net/http"

	githubsdk "github.com/google/go-github/v57/github"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/integrations/vcs/github"
	"github.com/hatchet-dev/hatchet/internal/repository"
)

func (g *GithubAppService) GithubUpdateTenantWebhook(ctx echo.Context, req gen.GithubUpdateTenantWebhookRequestObject) (gen.GithubUpdateTenantWebhookResponseObject, error) {
	webhookId := req.Webhook.String()

	webhook, err := g.config.APIRepository.Github().ReadGithubWebhookById(webhookId)

	if err != nil {
		return nil, err
	}

	signingSecret, err := g.config.Encryption.Decrypt(webhook.SigningSecret, "github_signing_secret")

	if err != nil {
		return nil, err
	}

	// validate the payload using the github webhook signing secret
	payload, err := githubsdk.ValidatePayload(ctx.Request(), signingSecret)

	if err != nil {
		return nil, err
	}

	event, err := githubsdk.ParseWebHook(githubsdk.WebHookType(ctx.Request()), payload)

	if err != nil {
		return nil, err
	}

	switch event := event.(type) { // nolint: gocritic
	case *githubsdk.PullRequestEvent:
		if err := g.processPullRequestEvent(webhook.TenantID, event, ctx.Request()); err != nil {
			return nil, fmt.Errorf("error processing pull request event: %w", err)
		}
	}

	return nil, nil
}

func (g *GithubAppService) processPullRequestEvent(tenantId string, event *githubsdk.PullRequestEvent, r *http.Request) error {
	pr := github.ToVCSRepositoryPullRequest(*event.GetRepo().GetOwner().Login, event.GetRepo().GetName(), event.GetPullRequest())

	dbPR, err := g.config.APIRepository.Github().GetPullRequest(tenantId, pr.GetRepoOwner(), pr.GetRepoName(), int(pr.GetPRNumber()))

	if err != nil {
		return err
	}

	_, err = g.config.APIRepository.Github().UpdatePullRequest(tenantId, dbPR.ID, &repository.UpdatePullRequestOpts{
		HeadBranch: repository.StringPtr(pr.GetHeadBranch()),
		BaseBranch: repository.StringPtr(pr.GetBaseBranch()),
		Title:      repository.StringPtr(pr.GetTitle()),
		State:      repository.StringPtr(pr.GetState()),
	})

	return err
}
