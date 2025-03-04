package workflowruns

import (
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *V1WorkflowRunsService) WithDags(ctx echo.Context, request gen.V1WorkflowRunListRequestObject) (gen.V1WorkflowRunListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	var (
		statuses = []sqlcv1.V1ReadableStatusOlap{
			sqlcv1.V1ReadableStatusOlapQUEUED,
			sqlcv1.V1ReadableStatusOlapRUNNING,
			sqlcv1.V1ReadableStatusOlapFAILED,
			sqlcv1.V1ReadableStatusOlapCOMPLETED,
			sqlcv1.V1ReadableStatusOlapCANCELLED,
		}
		since             = request.Params.Since
		workflowIds       = []uuid.UUID{}
		limit       int64 = 50
		offset      int64
	)

	if request.Params.Statuses != nil {
		if len(*request.Params.Statuses) > 0 {
			statuses = []sqlcv1.V1ReadableStatusOlap{}
			for _, status := range *request.Params.Statuses {
				statuses = append(statuses, sqlcv1.V1ReadableStatusOlap(status))
			}
		}
	}

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	if request.Params.WorkflowIds != nil {
		workflowIds = *request.Params.WorkflowIds
	}

	opts := v1.ListWorkflowRunOpts{
		CreatedAfter: since,
		Statuses:     statuses,
		WorkflowIds:  workflowIds,
		Limit:        limit,
		Offset:       offset,
	}

	additionalMetadataFilters := make(map[string]interface{})

	if request.Params.AdditionalMetadata != nil {
		for _, v := range *request.Params.AdditionalMetadata {
			kv_pairs := strings.Split(v, ":")
			if len(kv_pairs) == 2 {
				additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
			}
		}

		opts.AdditionalMetadata = additionalMetadataFilters
	}

	if request.Params.Until != nil {
		opts.FinishedBefore = request.Params.Until
	}

	if request.Params.ParentTaskExternalId != nil {
		parentTaskExternalId := request.Params.ParentTaskExternalId.String()
		id := sqlchelpers.UUIDFromStr(parentTaskExternalId)
		opts.ParentTaskExternalId = &id
	}

	dags, total, err := t.config.V1.OLAP().ListWorkflowRuns(
		ctx.Request().Context(),
		tenantId,
		opts,
	)

	if err != nil {
		return nil, err
	}

	dagExternalIds := make([]pgtype.UUID, 0)

	for _, dag := range dags {
		if dag.Kind == sqlcv1.V1RunKindDAG {
			dagExternalIds = append(dagExternalIds, dag.ExternalID)
		}
	}

	tasks, taskIdToDagExternalId, err := t.config.V1.OLAP().ListTasksByDAGId(
		ctx.Request().Context(),
		tenantId,
		dagExternalIds,
	)

	if err != nil {
		return nil, err
	}

	parsedTasks := transformers.TaskRunDataRowToWorkflowRunsMany(tasks, total, limit, offset)

	dagChildren := make(map[uuid.UUID][]gen.V1TaskSummary)

	for _, task := range parsedTasks.Rows {
		dagExternalId := taskIdToDagExternalId[int64(task.TaskId)]
		existing, ok := dagChildren[dagExternalId]

		if ok {
			dagChildren[dagExternalId] = append(existing, task)
		} else {
			dagChildren[dagExternalId] = []gen.V1TaskSummary{task}
		}
	}

	result := transformers.ToWorkflowRunMany(dags, dagChildren, total, limit, offset)

	// Search for api errors to see how we handle errors in other cases
	return gen.V1WorkflowRunList200JSONResponse(
		result,
	), nil
}

func (t *V1WorkflowRunsService) OnlyTasks(ctx echo.Context, request gen.V1WorkflowRunListRequestObject) (gen.V1WorkflowRunListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	var (
		statuses = []sqlcv1.V1ReadableStatusOlap{
			sqlcv1.V1ReadableStatusOlapQUEUED,
			sqlcv1.V1ReadableStatusOlapRUNNING,
			sqlcv1.V1ReadableStatusOlapFAILED,
			sqlcv1.V1ReadableStatusOlapCOMPLETED,
			sqlcv1.V1ReadableStatusOlapCANCELLED,
		}
		since             = request.Params.Since
		workflowIds       = []uuid.UUID{}
		limit       int64 = 50
		offset      int64
	)

	if request.Params.Statuses != nil {
		if len(*request.Params.Statuses) > 0 {
			statuses = []sqlcv1.V1ReadableStatusOlap{}
			for _, status := range *request.Params.Statuses {
				statuses = append(statuses, sqlcv1.V1ReadableStatusOlap(status))
			}
		}
	}

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	if request.Params.WorkflowIds != nil {
		workflowIds = *request.Params.WorkflowIds
	}

	opts := v1.ListTaskRunOpts{
		CreatedAfter: since,
		Statuses:     statuses,
		WorkflowIds:  workflowIds,
		Limit:        limit,
		Offset:       offset,
		WorkerId:     request.Params.WorkerId,
	}

	additionalMetadataFilters := make(map[string]interface{})

	if request.Params.AdditionalMetadata != nil {
		for _, v := range *request.Params.AdditionalMetadata {
			kv_pairs := strings.Split(v, ":")
			if len(kv_pairs) == 2 {
				additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
			}
		}

		opts.AdditionalMetadata = additionalMetadataFilters
	}

	if request.Params.Until != nil {
		opts.FinishedBefore = request.Params.Until
	}

	tasks, total, err := t.config.V1.OLAP().ListTasks(
		ctx.Request().Context(),
		tenantId,
		opts,
	)

	if err != nil {
		return nil, err
	}

	result := transformers.TaskRunDataRowToWorkflowRunsMany(tasks, total, limit, offset)

	// Search for api errors to see how we handle errors in other cases
	return gen.V1WorkflowRunList200JSONResponse(
		result,
	), nil
}

func (t *V1WorkflowRunsService) V1WorkflowRunList(ctx echo.Context, request gen.V1WorkflowRunListRequestObject) (gen.V1WorkflowRunListResponseObject, error) {
	if request.Params.OnlyTasks {
		return t.OnlyTasks(ctx, request)
	} else {
		return t.WithDags(ctx, request)
	}
}
