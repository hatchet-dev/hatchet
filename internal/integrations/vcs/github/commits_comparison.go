package github

import (
	githubsdk "github.com/google/go-github/v57/github"

	"github.com/hatchet-dev/hatchet/internal/integrations/vcs"
)

type GithubCommitsComparison struct {
	*githubsdk.CommitsComparison
}

func (g *GithubCommitsComparison) GetFiles() []vcs.CommitFile {
	var res []vcs.CommitFile

	for _, f := range g.CommitsComparison.Files {
		res = append(res, vcs.CommitFile{
			Name: f.GetFilename(),
		})
	}

	return res
}
