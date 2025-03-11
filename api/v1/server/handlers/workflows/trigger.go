package workflows

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkflowService) WorkflowRunCreate(ctx echo.Context, request gen.WorkflowRunCreateRequestObject) (gen.WorkflowRunCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	workflow := ctx.Get("workflow").(*dbsqlc.GetWorkflowByIdRow)

	var workflowVersionId string

	if request.Params.Version != nil {
		workflowVersionId = request.Params.Version.String()
	} else {

		if !workflow.WorkflowVersionId.Valid {
			return gen.WorkflowRunCreate400JSONResponse(
				apierrors.NewAPIErrors("workflow has no versions"),
			), nil
		}

		workflowVersionId = sqlchelpers.UUIDToStr(workflow.WorkflowVersionId)
	}

	workflowVersion, err := t.config.EngineRepository.Workflow().GetWorkflowVersionById(ctx.Request().Context(), tenantId, workflowVersionId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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

	var additionalMetadata map[string]interface{}

	if request.Body.AdditionalMetadata != nil {

		additionalMetadataBytes, err := json.Marshal(request.Body.AdditionalMetadata)
		if err != nil {
			return gen.WorkflowRunCreate400JSONResponse(
				apierrors.NewAPIErrors("Invalid additional metadata"),
			), nil
		}

		err = json.Unmarshal(additionalMetadataBytes, &additionalMetadata)
		if err != nil {
			return gen.WorkflowRunCreate400JSONResponse(
				apierrors.NewAPIErrors("Invalid additional metadata"),
			), nil
		}
	}

	createOpts, err := repository.GetCreateWorkflowRunOptsFromManual(workflowVersion, inputBytes, additionalMetadata)
	if err != nil {
		return nil, err
	}

	createdWorkflowRun, err := t.config.APIRepository.WorkflowRun().CreateNewWorkflowRun(ctx.Request().Context(), tenantId, createOpts)

	if err == metered.ErrResourceExhausted {
		return gen.WorkflowRunCreate429JSONResponse(
			apierrors.NewAPIErrors("Workflow Run limit exceeded"),
		), nil
	}

	if err != nil {
		return nil, fmt.Errorf("trigger.go could not create workflow run: %w", err)
	}

	// send to workflow processing queue
	err = t.config.MessageQueue.AddMessage(
		ctx.Request().Context(),
		msgqueue.WORKFLOW_PROCESSING_QUEUE,
		tasktypes.WorkflowRunQueuedToTask(
			sqlchelpers.UUIDToStr(createdWorkflowRun.TenantId),
			sqlchelpers.UUIDToStr(createdWorkflowRun.ID),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("could not add workflow run to queue: %w", err)
	}

	workflowRun, err := t.config.APIRepository.WorkflowRun().GetWorkflowRunById(ctx.Request().Context(), tenantId, sqlchelpers.UUIDToStr(createdWorkflowRun.ID))

	if err != nil {
		return nil, fmt.Errorf("could not get workflow run: %w", err)
	}

	res, err := transformers.ToWorkflowRun(workflowRun, nil, nil, nil)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowRunCreate200JSONResponse(
		*res,
	), nil
}
