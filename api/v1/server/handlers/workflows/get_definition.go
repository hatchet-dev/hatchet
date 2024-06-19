package workflows

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *WorkflowService) WorkflowVersionGetDefinition(ctx echo.Context, request gen.WorkflowVersionGetDefinitionRequestObject) (gen.WorkflowVersionGetDefinitionResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	workflow := ctx.Get("workflow").(*db.WorkflowModel)

	var workflowVersionId string

	if request.Params.Version != nil {
		workflowVersionId = request.Params.Version.String()
	} else {
		versions := workflow.Versions()

		if len(versions) == 0 {
			return gen.WorkflowVersionGetDefinition400JSONResponse(
				apierrors.NewAPIErrors("workflow has no versions"),
			), nil
		}

		workflowVersionId = versions[0].ID
	}

	workflowVersion, err := t.config.APIRepository.Workflow().GetWorkflowVersionById(tenant.ID, workflowVersionId)

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return gen.WorkflowVersionGetDefinition404JSONResponse(
				apierrors.NewAPIErrors("version not found"),
			), nil
		}

		return nil, err
	}

	rawDefinition, err := transformers.ToWorkflowYAMLBytes(workflow, workflowVersion)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowVersionGetDefinition200JSONResponse(gen.WorkflowVersionDefinition{
		RawDefinition: string(rawDefinition),
	}), nil
}
