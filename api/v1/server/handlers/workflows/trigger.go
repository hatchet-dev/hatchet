package workflows

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

func (t *WorkflowService) WorkflowRunCreate(ctx echo.Context, request gen.WorkflowRunCreateRequestObject) (gen.WorkflowRunCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	workflow := ctx.Get("workflow").(*db.WorkflowModel)

	var workflowVersionId string

	if request.Params.Version != nil {
		workflowVersionId = request.Params.Version.String()
	} else {
		versions := workflow.Versions()

		if len(versions) == 0 {
			return gen.WorkflowRunCreate400JSONResponse(
				apierrors.NewAPIErrors("workflow has no versions"),
			), nil
		}

		workflowVersionId = versions[0].ID
	}

	workflowVersion, err := t.config.Repository.Workflow().GetWorkflowVersionById(tenant.ID, workflowVersionId)

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return gen.WorkflowRunCreate400JSONResponse(
				apierrors.NewAPIErrors("version not found"),
			), nil
		}

		return nil, err
	}

	// make sure input can be marshalled and unmarshalled to input type
	inputBytes, err := json.Marshal(request.Body.Input)

	if err != nil {
		return gen.WorkflowRunCreate400JSONResponse(
			apierrors.NewAPIErrors("Invalid input"),
		), nil
	}

	createOpts, err := repository.GetCreateWorkflowRunOptsFromManual(workflowVersion, inputBytes)

	if err != nil {
		return nil, err
	}

	workflowRun, err := t.config.Repository.WorkflowRun().CreateNewWorkflowRun(ctx.Request().Context(), tenant.ID, createOpts)

	if err != nil {
		return nil, fmt.Errorf("could not create workflow run: %w", err)
	}

	// send to workflow processing queue
	err = t.config.TaskQueue.AddTask(
		ctx.Request().Context(),
		taskqueue.WORKFLOW_PROCESSING_QUEUE,
		tasktypes.WorkflowRunQueuedToTask(workflowRun),
	)

	if err != nil {
		return nil, fmt.Errorf("could not add workflow run to queue: %w", err)
	}

	res, err := transformers.ToWorkflowRun(workflowRun)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowRunCreate200JSONResponse(
		*res,
	), nil
}
