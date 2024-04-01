package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (t *WorkflowService) WorkflowRunListPullRequests(ctx echo.Context, request gen.WorkflowRunListPullRequestsRequestObject) (gen.WorkflowRunListPullRequestsResponseObject, error) {
	workflowRun := ctx.Get("workflow-run").(*db.WorkflowRunModel)

	listOpts := &repository.ListPullRequestsForWorkflowRunOpts{}

	if request.Params.State != nil {
		listOpts.State = repository.StringPtr(string(*request.Params.State))
	}

	prs, err := t.config.APIRepository.WorkflowRun().ListPullRequestsForWorkflowRun(
		workflowRun.TenantID,
		workflowRun.ID,
		listOpts,
	)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.PullRequest, 0)

	for _, pr := range prs {
		prCp := pr
		rows = append(rows, *transformers.ToPullRequest(&prCp))
	}

	return gen.WorkflowRunListPullRequests200JSONResponse(gen.ListPullRequestsResponse{
		PullRequests: rows,
	}), nil
}
