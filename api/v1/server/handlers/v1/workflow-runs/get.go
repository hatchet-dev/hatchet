package workflowruns

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *V1WorkflowRunsService) V1WorkflowRunGet(ctx echo.Context, request gen.V1WorkflowRunGetRequestObject) (gen.V1WorkflowRunGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	rawWorkflowRun := ctx.Get("v2-workflow-run").(*v1.V1WorkflowRunPopulator)

	workflowRun := rawWorkflowRun.WorkflowRun
	taskMetadata := rawWorkflowRun.TaskMetadata

	workflowRunId := workflowRun.ExternalID

	requestContext := ctx.Request().Context()

	taskRunEvents, err := t.config.V1.OLAP().ListTaskRunEventsByWorkflowRunId(
		requestContext,
		tenantId,
		workflowRunId,
	)

	if err != nil {
		return nil, err
	}

	tasks, err := t.config.V1.OLAP().ListTasksByIdAndInsertedAt(
		requestContext,
		tenantId,
		taskMetadata,
	)

	if err != nil {
		return nil, err
	}

	stepIdToTaskExternalId := make(map[pgtype.UUID]pgtype.UUID)
	for _, task := range tasks {
		stepIdToTaskExternalId[task.StepID] = task.ExternalID
	}

	workflowVersionId := uuid.MustParse(sqlchelpers.UUIDToStr(workflowRun.WorkflowVersionId))

	shape, err := t.config.APIRepository.WorkflowRun().GetWorkflowRunShape(
		requestContext, workflowVersionId,
	)

	if err != nil {
		return nil, err
	}

	result, err := transformers.ToWorkflowRunDetails(taskRunEvents, workflowRun, shape, tasks, stepIdToTaskExternalId)

	if err != nil {
		return nil, err
	}

	// Search for api errors to see how we handle errors in other cases
	return gen.V1WorkflowRunGet200JSONResponse(
		result,
	), nil
}
