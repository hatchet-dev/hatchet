package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"

	githubsdk "github.com/google/go-github/v57/github"

	"github.com/hatchet-dev/hatchet/internal/encryption"
	"github.com/hatchet-dev/hatchet/internal/integrations/vcs"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

type GithubVCSProvider struct {
	repo      repository.Repository
	appConf   *GithubAppConf
	serverURL string
	enc       encryption.EncryptionService
}

func NewGithubVCSProvider(appConf *GithubAppConf, repo repository.Repository, serverURL string, enc encryption.EncryptionService) GithubVCSProvider {
	return GithubVCSProvider{
		appConf:   appConf,
		repo:      repo,
		serverURL: serverURL,
		enc:       enc,
	}
}

func ToGithubVCSProvider(provider vcs.VCSProvider) (res GithubVCSProvider, err error) {
	res, ok := provider.(GithubVCSProvider)

	if !ok {
		return res, fmt.Errorf("could not convert VCS provider to Github VCS provider: %w", err)
	}

	return res, nil
}

func (g GithubVCSProvider) GetGithubAppConfig() *GithubAppConf {
	return g.appConf
}

func (g GithubVCSProvider) GetVCSRepositoryFromWorkflow(workflow *db.WorkflowModel) (vcs.VCSRepository, error) {
	var installationId string
	var deploymentConf *db.WorkflowDeploymentConfigModel
	var ok bool

	if deploymentConf, ok = workflow.DeploymentConfig(); ok {
		if installationId, ok = deploymentConf.GithubAppInstallationID(); !ok {
			return nil, fmt.Errorf("module does not have github app installation id param set")
		}
	}

	gai, err := g.repo.Github().ReadGithubAppInstallationByID(installationId)

	if err != nil {
		return nil, err
	}

	client, err := g.appConf.GetGithubClient(int64(gai.InstallationID))

	if err != nil {
		return nil, err
	}

	return &GithubVCSRepository{
		repoOwner: deploymentConf.GitRepoOwner,
		repoName:  deploymentConf.GitRepoName,
		serverURL: g.serverURL,
		client:    client,
		repo:      g.repo,
		enc:       g.enc,
	}, nil
}

// func (g GithubVCSProvider) GetVCSRepositoryFromGAI(gai *models.GithubAppInstallation) (vcs.VCSRepository, error) {
// }

type GithubVCSRepository struct {
	repoOwner, repoName string
	client              *githubsdk.Client
	repo                repository.Repository
	serverURL           string
	enc                 encryption.EncryptionService
}

// GetKind returns the kind of VCS provider -- used for downstream integrations
func (g *GithubVCSRepository) GetKind() vcs.VCSRepositoryKind {
	return vcs.VCSRepositoryKindGithub
}

func (g *GithubVCSRepository) GetRepoOwner() string {
	return g.repoOwner
}

func (g *GithubVCSRepository) GetRepoName() string {
	return g.repoName
}

// SetupRepository sets up a VCS repository on Hatchet.
func (g *GithubVCSRepository) SetupRepository(tenantId string) error {
	repoOwner := g.GetRepoOwner()
	repoName := g.GetRepoName()

	_, err := g.repo.Github().ReadGithubWebhook(tenantId, repoOwner, repoName)

	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return err
	} else if err != nil {
		opts, signingSecret, err := repository.NewGithubWebhookCreateOpts(g.enc, repoOwner, repoName)

		if err != nil {
			return err
		}

		gw, err := g.repo.Github().CreateGithubWebhook(tenantId, opts)

		if err != nil {
			return err
		}

		webhookURL := fmt.Sprintf("%s/api/v1/teams/%s/github_incoming/%s", g.serverURL, tenantId, gw.ID)

		_, _, err = g.client.Repositories.CreateHook(
			context.Background(), repoOwner, repoName, &githubsdk.Hook{
				Config: map[string]interface{}{
					"url":          webhookURL,
					"content_type": "json",
					"secret":       signingSecret,
				},
				Events: []string{"pull_request", "push"},
				Active: githubsdk.Bool(true),
			},
		)

		return err
	}

	return nil
}

// GetArchiveLink returns an archive link for a specific repo SHA
func (g *GithubVCSRepository) GetArchiveLink(ref string) (*url.URL, error) {
	gURL, _, err := g.client.Repositories.GetArchiveLink(
		context.TODO(),
		g.GetRepoOwner(),
		g.GetRepoName(),
		githubsdk.Zipball,
		&githubsdk.RepositoryContentGetOptions{
			Ref: ref,
		},
		2,
	)

	return gURL, err
}

// GetBranch gets a full branch (name and sha)
func (g *GithubVCSRepository) GetBranch(name string) (vcs.VCSBranch, error) {
	branchResp, _, err := g.client.Repositories.GetBranch(
		context.TODO(),
		g.GetRepoOwner(),
		g.GetRepoName(),
		name,
		2,
	)

	if err != nil {
		return nil, err
	}

	return &GithubBranch{branchResp}, nil
}

// ReadFile returns a file by a SHA reference or path
func (g *GithubVCSRepository) ReadFile(ref, path string) (io.ReadCloser, error) {
	file, _, err := g.client.Repositories.DownloadContents(
		context.Background(),
		g.GetRepoOwner(),
		g.GetRepoName(),
		path,
		&githubsdk.RepositoryContentGetOptions{
			Ref: ref,
		},
	)

	return file, err
}

func (g *GithubVCSRepository) ReadDirectory(ref, path string) ([]vcs.DirectoryItem, error) {
	_, dirs, _, err := g.client.Repositories.GetContents(
		context.Background(),
		g.GetRepoOwner(),
		g.GetRepoName(),
		path,
		&githubsdk.RepositoryContentGetOptions{
			Ref: ref,
		},
	)

	if err != nil {
		return nil, err
	}

	res := []vcs.DirectoryItem{}

	for _, item := range dirs {
		if item.Type != nil && item.Name != nil {
			res = append(res, vcs.DirectoryItem{
				Type: *item.Type,
				Name: *item.Name,
			})
		}
	}

	return res, nil
}

func (g *GithubVCSRepository) CreateOrUpdatePullRequest(tenantId, workflowRunId string, opts *vcs.CreatePullRequestOpts) (*db.GithubPullRequestModel, error) {
	// determine if there's an open pull request for this workflow run
	prs, err := g.repo.WorkflowRun().ListPullRequestsForWorkflowRun(tenantId, workflowRunId, &repository.ListPullRequestsForWorkflowRunOpts{
		State: repository.StringPtr("open"),
	})

	if err != nil {
		return nil, err
	}

	if len(prs) > 0 {
		// double check that the PR is still open, cycle through PRs to find the first open one
		for _, pr := range prs {
			prCp := pr

			ghPR, err := g.getPullRequest(tenantId, workflowRunId, &prCp)

			if err != nil {
				return nil, err
			}

			if prCp.PullRequestState != ghPR.GetState() {
				defer g.repo.Github().UpdatePullRequest(tenantId, prCp.ID, &repository.UpdatePullRequestOpts{ // nolint: errcheck
					State: repository.StringPtr(ghPR.GetState()),
				})
			}

			if ghPR.GetState() == "open" {
				return g.updatePullRequest(tenantId, workflowRunId, &prCp, opts)
			}
		}
	}

	// if we get here, we need to create a new PR
	return g.createPullRequest(tenantId, workflowRunId, opts)
}

func (g *GithubVCSRepository) getPullRequest(tenantId, workflowRunId string, pr *db.GithubPullRequestModel) (*githubsdk.PullRequest, error) {
	ghPR, _, err := g.client.PullRequests.Get(
		context.Background(),
		pr.RepositoryOwner,
		pr.RepositoryName,
		pr.PullRequestNumber,
	)

	return ghPR, err
}

func (g *GithubVCSRepository) updatePullRequest(tenantId, workflowRunId string, pr *db.GithubPullRequestModel, opts *vcs.CreatePullRequestOpts) (*db.GithubPullRequestModel, error) {
	err := commitFiles(
		g.client,
		opts.Files,
		opts.GitRepoOwner,
		opts.GitRepoName,
		opts.HeadBranchName,
	)

	if err != nil {
		return nil, fmt.Errorf("Could not commit files: %w", err)
	}

	return pr, nil
}

func (g *GithubVCSRepository) createPullRequest(tenantId, workflowRunId string, opts *vcs.CreatePullRequestOpts) (*db.GithubPullRequestModel, error) {
	var baseBranch string

	if opts.BaseBranch == nil {
		repo, _, err := g.client.Repositories.Get(
			context.TODO(),
			opts.GitRepoOwner,
			opts.GitRepoName,
		)

		if err != nil {
			return nil, err
		}

		baseBranch = repo.GetDefaultBranch()
	}

	err := createNewBranch(g.client, opts.GitRepoOwner, opts.GitRepoName, baseBranch, opts.HeadBranchName)

	if err != nil {
		return nil, fmt.Errorf("Could not create PR: %w", err)
	}

	err = commitFiles(
		g.client,
		opts.Files,
		opts.GitRepoOwner,
		opts.GitRepoName,
		opts.HeadBranchName,
	)

	if err != nil {
		return nil, fmt.Errorf("Could not commit files: %w", err)
	}

	pr, _, err := g.client.PullRequests.Create(
		context.Background(), opts.GitRepoOwner, opts.GitRepoName, &githubsdk.NewPullRequest{
			Title: githubsdk.String(opts.Title),
			Base:  githubsdk.String(baseBranch),
			Head:  githubsdk.String(opts.HeadBranchName),
		},
	)

	if err != nil {
		return nil, err
	}

	return g.repo.WorkflowRun().CreateWorkflowRunPullRequest(tenantId, workflowRunId, &repository.CreateWorkflowRunPullRequestOpts{
		RepositoryOwner:       opts.GitRepoOwner,
		RepositoryName:        opts.GitRepoName,
		PullRequestID:         int(pr.GetID()),
		PullRequestTitle:      opts.Title,
		PullRequestNumber:     pr.GetNumber(),
		PullRequestHeadBranch: opts.HeadBranchName,
		PullRequestBaseBranch: baseBranch,
		PullRequestState:      pr.GetState(),
	})
}

// CompareCommits compares a base commit with a head commit
func (g *GithubVCSRepository) CompareCommits(base, head string) (vcs.VCSCommitsComparison, error) {
	commitsRes, _, err := g.client.Repositories.CompareCommits(
		context.Background(),
		g.GetRepoOwner(),
		g.GetRepoName(),
		base,
		head,
		&githubsdk.ListOptions{},
	)

	if err != nil {
		return nil, err
	}

	return &GithubCommitsComparison{commitsRes}, nil
}
