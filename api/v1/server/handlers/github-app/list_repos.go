package githubapp

import (
	"context"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	githubsdk "github.com/google/go-github/v57/github"
)

func (g *GithubAppService) GithubAppListRepos(ctx echo.Context, req gen.GithubAppListReposRequestObject) (gen.GithubAppListReposResponseObject, error) {
	client, reqErr := GetGithubAppClientFromRequest(ctx, g.config)

	if reqErr != nil {
		return gen.GithubAppListRepos400JSONResponse(
			*reqErr,
		), nil
	}

	// figure out number of repositories
	opt := &githubsdk.ListOptions{
		PerPage: 100,
	}

	repoList, resp, err := client.Apps.ListRepos(context.Background(), opt)

	if err != nil {
		return nil, err
	}

	allRepos := repoList.Repositories

	// make workers to get pages concurrently
	const WCOUNT = 5
	numPages := resp.LastPage + 1
	var workerErr error
	var mu sync.Mutex
	var wg sync.WaitGroup

	worker := func(cp int) {
		defer wg.Done()

		for cp < numPages {
			cur_opt := &githubsdk.ListOptions{
				Page:    cp,
				PerPage: 100,
			}

			repos, _, err := client.Apps.ListRepos(context.Background(), cur_opt)

			if err != nil {
				mu.Lock()
				workerErr = err
				mu.Unlock()
				return
			}

			mu.Lock()
			allRepos = append(allRepos, repos.Repositories...)
			mu.Unlock()

			cp += WCOUNT
		}
	}

	var numJobs int
	if numPages > WCOUNT {
		numJobs = WCOUNT
	} else {
		numJobs = numPages
	}

	wg.Add(numJobs)

	// page 1 is already loaded so we start with 2
	for i := 1; i <= numJobs; i++ {
		go worker(i + 1)
	}

	wg.Wait()

	if workerErr != nil {
		return nil, workerErr
	}

	res := make([]gen.GithubRepo, 0)

	for _, repo := range allRepos {
		res = append(res, gen.GithubRepo{
			RepoName:  *repo.Name,
			RepoOwner: *repo.Owner.Login,
		})
	}

	return gen.GithubAppListRepos200JSONResponse(
		res,
	), nil
}
