package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/integrations/vcs"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *WorkflowService) WorkflowUpdateLinkGithub(ctx echo.Context, request gen.WorkflowUpdateLinkGithubRequestObject) (gen.WorkflowUpdateLinkGithubResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)
	workflow := ctx.Get("workflow").(*db.WorkflowModel)

	// check that the user has access to the installation id
	installationId := request.Body.InstallationId

	_, err := t.config.APIRepository.Github().ReadGithubAppInstallationByID(installationId)

	if err != nil {
		return gen.WorkflowUpdateLinkGithub404JSONResponse(
			apierrors.NewAPIErrors("Installation not found"),
		), nil
	}

	if canAccess, err := t.config.APIRepository.Github().CanUserAccessInstallation(installationId, user.ID); err != nil || !canAccess {
		return gen.WorkflowUpdateLinkGithub403JSONResponse(
			apierrors.NewAPIErrors("User does not have access to the installation"),
		), nil
	}

	_, err = t.config.APIRepository.Workflow().UpsertWorkflowDeploymentConfig(
		workflow.ID,
		&repository.UpsertWorkflowDeploymentConfigOpts{
			GithubAppInstallationId: installationId,
			GitRepoName:             request.Body.GitRepoName,
			GitRepoOwner:            request.Body.GitRepoOwner,
			GitRepoBranch:           request.Body.GitRepoBranch,
		},
	)

	if err != nil {
		return nil, err
	}

	workflow, err = t.config.APIRepository.Workflow().GetWorkflowById(workflow.ID)

	if err != nil {
		return nil, err
	}

	vcsProvider := t.config.VCSProviders[vcs.VCSRepositoryKindGithub]
	vcs, err := vcsProvider.GetVCSRepositoryFromWorkflow(workflow)

	if err != nil {
		return nil, err
	}

	err = vcs.SetupRepository(workflow.TenantID)

	if err != nil {
		return nil, err
	}

	resp, err := transformers.ToWorkflow(workflow, nil)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowUpdateLinkGithub200JSONResponse(*resp), nil
}
