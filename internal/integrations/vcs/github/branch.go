package github

import (
	githubsdk "github.com/google/go-github/v57/github"
)

type GithubBranch struct {
	*githubsdk.Branch
}

func (g *GithubBranch) GetName() string {
	return g.Branch.GetName()
}

func (g *GithubBranch) GetLatestRef() string {
	return g.GetCommit().GetSHA()
}
