package prisma

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type githubRepository struct {
	client *db.PrismaClient
	v      validator.Validator
}

func NewGithubRepository(client *db.PrismaClient, v validator.Validator) repository.GithubRepository {
	return &githubRepository{
		client: client,
		v:      v,
	}
}

func (r *githubRepository) CreateInstallation(githubUserId int, opts *repository.CreateInstallationOpts) (*db.GithubAppInstallationModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.GithubAppInstallation.CreateOne(
		db.GithubAppInstallation.GithubAppOAuth.Link(
			db.GithubAppOAuth.GithubUserID.Equals(githubUserId),
		),
		db.GithubAppInstallation.InstallationID.Set(opts.InstallationID),
		db.GithubAppInstallation.AccountName.Set(opts.AccountName),
		db.GithubAppInstallation.AccountID.Set(opts.AccountID),
		db.GithubAppInstallation.Config.SetIfPresent(opts.Config),
		db.GithubAppInstallation.AccountAvatarURL.SetIfPresent(opts.AccountAvatarURL),
		db.GithubAppInstallation.InstallationSettingsURL.SetIfPresent(opts.InstallationSettingsURL),
	).Exec(context.Background())
}

func (r *githubRepository) AddGithubUserIdToInstallation(installationID, accountID, githubUserId int) (*db.GithubAppInstallationModel, error) {
	return r.client.GithubAppInstallation.FindUnique(
		db.GithubAppInstallation.InstallationIDAccountID(
			db.GithubAppInstallation.InstallationID.Equals(installationID),
			db.GithubAppInstallation.AccountID.Equals(accountID),
		),
	).Update(
		db.GithubAppInstallation.GithubAppOAuth.Link(
			db.GithubAppOAuth.GithubUserID.Equals(githubUserId),
		),
	).Exec(context.Background())
}

func (r *githubRepository) ReadGithubAppInstallationByID(installationId string) (*db.GithubAppInstallationModel, error) {
	return r.client.GithubAppInstallation.FindUnique(
		db.GithubAppInstallation.ID.Equals(installationId),
	).Exec(context.Background())
}

func (r *githubRepository) ReadGithubWebhook(tenantId, repoOwner, repoName string) (*db.GithubWebhookModel, error) {
	return r.client.GithubWebhook.FindUnique(
		db.GithubWebhook.TenantIDRepositoryOwnerRepositoryName(
			db.GithubWebhook.TenantID.Equals(tenantId),
			db.GithubWebhook.RepositoryOwner.Equals(repoOwner),
			db.GithubWebhook.RepositoryName.Equals(repoName),
		),
	).Exec(context.Background())
}

func (r *githubRepository) ReadGithubWebhookById(id string) (*db.GithubWebhookModel, error) {
	return r.client.GithubWebhook.FindUnique(
		db.GithubWebhook.ID.Equals(id),
	).Exec(context.Background())
}

func (r *githubRepository) ReadGithubAppOAuthByGithubUserID(githubUserId int) (*db.GithubAppOAuthModel, error) {
	return r.client.GithubAppOAuth.FindUnique(
		db.GithubAppOAuth.GithubUserID.Equals(githubUserId),
	).Exec(context.Background())
}

func (r *githubRepository) ReadGithubAppInstallationByInstallationAndAccountID(installationID, accountID int) (*db.GithubAppInstallationModel, error) {
	return r.client.GithubAppInstallation.FindUnique(
		db.GithubAppInstallation.InstallationIDAccountID(
			db.GithubAppInstallation.InstallationID.Equals(installationID),
			db.GithubAppInstallation.AccountID.Equals(accountID),
		),
	).Exec(context.Background())
}

func (r *githubRepository) CreateGithubWebhook(tenantId string, opts *repository.CreateGithubWebhookOpts) (*db.GithubWebhookModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.GithubWebhook.CreateOne(
		db.GithubWebhook.Tenant.Link(
			db.Tenant.ID.Equals(tenantId),
		),
		db.GithubWebhook.RepositoryOwner.Set(opts.RepoOwner),
		db.GithubWebhook.RepositoryName.Set(opts.RepoName),
		db.GithubWebhook.SigningSecret.Set(opts.SigningSecret),
	).Exec(context.Background())
}

func (r *githubRepository) UpsertGithubAppOAuth(userId string, opts *repository.CreateGithubAppOAuthOpts) (*db.GithubAppOAuthModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.GithubAppOAuth.UpsertOne(
		db.GithubAppOAuth.GithubUserID.Equals(opts.GithubUserID),
	).Create(
		db.GithubAppOAuth.GithubUserID.Set(opts.GithubUserID),
		db.GithubAppOAuth.AccessToken.Set(opts.AccessToken),
		db.GithubAppOAuth.RefreshToken.SetIfPresent(opts.RefreshToken),
		db.GithubAppOAuth.ExpiresAt.SetIfPresent(opts.ExpiresAt),
		db.GithubAppOAuth.Users.Link(
			db.User.ID.Equals(userId),
		),
	).Update(
		db.GithubAppOAuth.AccessToken.Set(opts.AccessToken),
		db.GithubAppOAuth.RefreshToken.SetIfPresent(opts.RefreshToken),
		db.GithubAppOAuth.ExpiresAt.SetIfPresent(opts.ExpiresAt),
		db.GithubAppOAuth.Users.Link(
			db.User.ID.Equals(userId),
		),
	).Exec(context.Background())
}

func (r *githubRepository) CanUserAccessInstallation(installationId, userId string) (bool, error) {
	installation, err := r.client.GithubAppInstallation.FindFirst(
		db.GithubAppInstallation.ID.Equals(installationId),
		db.GithubAppInstallation.GithubAppOAuth.Where(
			db.GithubAppOAuth.Users.Some(
				db.User.ID.Equals(userId),
			),
		),
	).Exec(context.Background())

	if err != nil {
		return false, nil
	}

	return installation != nil, nil
}

func (r *githubRepository) DeleteInstallation(installationId, accountId int) (*db.GithubAppInstallationModel, error) {
	return r.client.GithubAppInstallation.FindUnique(
		db.GithubAppInstallation.InstallationIDAccountID(
			db.GithubAppInstallation.InstallationID.Equals(installationId),
			db.GithubAppInstallation.AccountID.Equals(accountId),
		),
	).Delete().Exec(context.Background())
}

func (r *githubRepository) ListGithubAppInstallationsByUserID(userId string) ([]db.GithubAppInstallationModel, error) {
	return r.client.GithubAppInstallation.FindMany(
		db.GithubAppInstallation.GithubAppOAuth.Where(
			db.GithubAppOAuth.Users.Some(
				db.User.ID.Equals(userId),
			),
		),
	).Exec(context.Background())
}

func (r *githubRepository) UpdatePullRequest(tenantId, prId string, opts *repository.UpdatePullRequestOpts) (*db.GithubPullRequestModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.GithubPullRequest.FindUnique(
		db.GithubPullRequest.ID.Equals(prId),
	).Update(
		db.GithubPullRequest.PullRequestState.SetIfPresent(opts.State),
		db.GithubPullRequest.PullRequestHeadBranch.SetIfPresent(opts.HeadBranch),
		db.GithubPullRequest.PullRequestBaseBranch.SetIfPresent(opts.BaseBranch),
		db.GithubPullRequest.PullRequestTitle.SetIfPresent(opts.Title),
	).Exec(context.Background())
}

func (r *githubRepository) GetPullRequest(tenantId, repoOwner, repoName string, prNumber int) (*db.GithubPullRequestModel, error) {
	return r.client.GithubPullRequest.FindUnique(
		db.GithubPullRequest.TenantIDRepositoryOwnerRepositoryNamePullRequestNumber(
			db.GithubPullRequest.TenantID.Equals(tenantId),
			db.GithubPullRequest.RepositoryOwner.Equals(repoOwner),
			db.GithubPullRequest.RepositoryName.Equals(repoName),
			db.GithubPullRequest.PullRequestNumber.Equals(prNumber),
		),
	).Exec(context.Background())
}
