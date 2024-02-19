package repository

import (
	"fmt"
	"time"

	"github.com/steebchen/prisma-client-go/runtime/types"

	"github.com/hatchet-dev/hatchet/internal/encryption"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

type CreateInstallationOpts struct {
	// (required) the installation id
	InstallationID int `validate:"required"`

	// (required) the account ID
	AccountID int `validate:"required"`

	// (required) the account name
	AccountName string `validate:"required"`

	// (optional) the account avatar URL
	AccountAvatarURL *string

	// (optional) the installation settings URL
	InstallationSettingsURL *string

	// (optional) the installation configuration
	Config *types.JSON
}

type CreateGithubWebhookOpts struct {
	// (required) the repo owner
	RepoOwner string `validate:"required"`

	// (required) the repo name
	RepoName string `validate:"required"`

	// (required) the signing secret
	SigningSecret []byte `validate:"required,min=1"`
}

func NewGithubWebhookCreateOpts(
	enc encryption.EncryptionService,
	repoOwner string,
	repoName string,
) (opts *CreateGithubWebhookOpts, signingSecret string, err error) {
	signingSecret, err = encryption.GenerateRandomBytes(16)

	if err != nil {
		return nil, "", fmt.Errorf("failed to generate signing secret: %s", err.Error())
	}

	// use the encryption service to encrypt the access and refresh token
	signingSecretEncrypted, err := enc.Encrypt([]byte(signingSecret), "github_signing_secret")

	if err != nil {
		return nil, "", fmt.Errorf("failed to encrypt access token: %s", err.Error())
	}

	opts = &CreateGithubWebhookOpts{
		RepoOwner:     repoOwner,
		RepoName:      repoName,
		SigningSecret: signingSecretEncrypted,
	}

	return opts, signingSecret, nil
}

type CreateGithubAppOAuthOpts struct {
	// (required) the oauth provider's user id
	GithubUserID int `validate:"required"`

	// (required) the oauth provider's access token
	AccessToken []byte `validate:"required,min=1"`

	// (optional) the oauth provider's refresh token
	RefreshToken *[]byte

	// (optional) the oauth provider's expiry time
	ExpiresAt *time.Time

	// (optional) the oauth provider's configuration
	Config *types.JSON
}

type UpdateInstallationOpts struct {
}

type UpdatePullRequestOpts struct {
	Title *string

	State *string

	HeadBranch *string

	BaseBranch *string
}

type GithubRepository interface {
	CreateInstallation(githubUserId int, opts *CreateInstallationOpts) (*db.GithubAppInstallationModel, error)

	AddGithubUserIdToInstallation(installationID, accountID, githubUserId int) (*db.GithubAppInstallationModel, error)

	ReadGithubAppInstallationByID(installationId string) (*db.GithubAppInstallationModel, error)

	ReadGithubWebhookById(id string) (*db.GithubWebhookModel, error)

	ReadGithubWebhook(tenantId, repoOwner, repoName string) (*db.GithubWebhookModel, error)

	ReadGithubAppOAuthByGithubUserID(githubUserID int) (*db.GithubAppOAuthModel, error)

	CanUserAccessInstallation(installationId, userId string) (bool, error)

	ReadGithubAppInstallationByInstallationAndAccountID(installationID, accountID int) (*db.GithubAppInstallationModel, error)

	CreateGithubWebhook(tenantId string, opts *CreateGithubWebhookOpts) (*db.GithubWebhookModel, error)

	UpsertGithubAppOAuth(userId string, opts *CreateGithubAppOAuthOpts) (*db.GithubAppOAuthModel, error)

	DeleteInstallation(installationId, accountId int) (*db.GithubAppInstallationModel, error)

	ListGithubAppInstallationsByUserID(userId string) ([]db.GithubAppInstallationModel, error)

	UpdatePullRequest(tenantId, prId string, opts *UpdatePullRequestOpts) (*db.GithubPullRequestModel, error)

	GetPullRequest(tenantId, repoOwner, repoName string, prNumber int) (*db.GithubPullRequestModel, error)
}
