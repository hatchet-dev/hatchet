package githubapp

import (
	"context"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	githubsdk "github.com/google/go-github/v57/github"
)

func (g *GithubAppService) GithubAppListBranches(ctx echo.Context, req gen.GithubAppListBranchesRequestObject) (gen.GithubAppListBranchesResponseObject, error) {
	owner := req.GhRepoOwner
	name := req.GhRepoName

	client, reqErr := GetGithubAppClientFromRequest(ctx, g.config)

	if reqErr != nil {
		return gen.GithubAppListBranches400JSONResponse(
			*reqErr,
		), nil
	}

	repo, _, err := client.Repositories.Get(
		context.TODO(),
		owner,
		name,
	)

	if err != nil {
		return nil, err
	}

	defaultBranch := repo.GetDefaultBranch()

	// List all branches for a specified repo
	allBranches, resp, err := client.Repositories.ListBranches(context.Background(), owner, name, &githubsdk.BranchListOptions{
		ListOptions: githubsdk.ListOptions{
			PerPage: 100,
		},
	})

	if err != nil {
		return nil, err
	}

	// make workers to get branches concurrently
	const WCOUNT = 5
	numPages := resp.LastPage + 1
	var workerErr error
	var mu sync.Mutex
	var wg sync.WaitGroup

	worker := func(cp int) {
		defer wg.Done()

		for cp < numPages {
			opts := &githubsdk.BranchListOptions{
				ListOptions: githubsdk.ListOptions{
					Page:    cp,
					PerPage: 100,
				},
			}

			branches, _, err := client.Repositories.ListBranches(context.Background(), owner, name, opts)

			if err != nil {
				mu.Lock()
				workerErr = err
				mu.Unlock()
				return
			}

			mu.Lock()
			allBranches = append(allBranches, branches...)
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

	res := make([]gen.GithubBranch, 0)

	for _, branch := range allBranches {
		res = append(res, gen.GithubBranch{
			BranchName: *branch.Name,
			IsDefault:  defaultBranch == *branch.Name,
		})
	}

	return gen.GithubAppListBranches200JSONResponse(
		res,
	), nil
}
