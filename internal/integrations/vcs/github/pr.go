package github

import (
	"context"
	"fmt"
	"net/http"

	githubsdk "github.com/google/go-github/v57/github"

	"github.com/hatchet-dev/hatchet/internal/integrations/vcs"
)

type ghPullRequest struct {
	repoOwner, repoName string
	pr                  *githubsdk.PullRequest
}

func ToVCSRepositoryPullRequest(repoOwner, repoName string, pr *githubsdk.PullRequest) vcs.VCSRepositoryPullRequest {
	return &ghPullRequest{repoOwner, repoName, pr}
}

func (g *ghPullRequest) GetRepoOwner() string {
	return g.repoOwner
}

func (g *ghPullRequest) GetRepoName() string {
	return g.repoName
}

func (g *ghPullRequest) GetVCSID() vcs.VCSObjectID {
	return vcs.NewVCSObjectInt(g.pr.GetID())
}

func (g *ghPullRequest) GetPRNumber() int64 {
	return int64(g.pr.GetNumber())
}

func (g *ghPullRequest) GetBaseSHA() string {
	return g.pr.GetBase().GetSHA()
}

func (g *ghPullRequest) GetHeadSHA() string {
	return g.pr.GetHead().GetSHA()
}

func (g *ghPullRequest) GetBaseBranch() string {
	return g.pr.GetBase().GetRef()
}

func (g *ghPullRequest) GetHeadBranch() string {
	return g.pr.GetHead().GetRef()
}

func (g *ghPullRequest) GetTitle() string {
	return g.pr.GetTitle()
}

func (g *ghPullRequest) GetState() string {
	return g.pr.GetState()
}

func createNewBranch(
	client *githubsdk.Client,
	gitRepoOwner, gitRepoName, baseBranch, headBranch string,
) error {
	_, resp, err := client.Repositories.GetBranch(
		context.Background(), gitRepoOwner, gitRepoName, headBranch, 2,
	)

	headBranchRef := fmt.Sprintf("refs/heads/%s", headBranch)

	if err == nil {
		return fmt.Errorf("branch %s already exists", headBranch)
	} else if resp.StatusCode != http.StatusNotFound {
		return err
	}

	base, _, err := client.Repositories.GetBranch(
		context.Background(), gitRepoOwner, gitRepoName, baseBranch, 2,
	)

	if err != nil {
		return err
	}

	_, _, err = client.Git.CreateRef(
		context.Background(), gitRepoOwner, gitRepoName, &githubsdk.Reference{
			Ref: githubsdk.String(headBranchRef),
			Object: &githubsdk.GitObject{
				SHA: base.Commit.SHA,
			},
		},
	)

	if err != nil {
		return err
	}

	return nil
}

func commitFiles(
	client *githubsdk.Client,
	files map[string][]byte,
	gitRepoOwner, gitRepoName, branch string,
) error {
	for filepath, contents := range files {
		sha := ""

		// get contents of a file if it exists
		fileData, _, _, _ := client.Repositories.GetContents(
			context.TODO(),
			gitRepoOwner,
			gitRepoName,
			filepath,
			&githubsdk.RepositoryContentGetOptions{
				Ref: branch,
			},
		)

		if fileData != nil {
			sha = *fileData.SHA
		}

		opts := &githubsdk.RepositoryContentFileOptions{
			Message: githubsdk.String(fmt.Sprintf("Create %s file", filepath)),
			Content: contents,
			Branch:  githubsdk.String(branch),
			SHA:     &sha,
		}

		opts.Committer = &githubsdk.CommitAuthor{
			Name:  githubsdk.String("Hatchet Bot"),
			Email: githubsdk.String("contact@hatchet.run"),
		}

		_, _, err := client.Repositories.UpdateFile(
			context.TODO(),
			gitRepoOwner,
			gitRepoName,
			filepath,
			opts,
		)

		if err != nil {
			return err
		}
	}

	return nil
}
