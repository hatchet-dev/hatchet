package tasks

import (
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1TaskList(ctx echo.Context, request gen.V1TaskListRequestObject) (gen.V1TaskListResponseObject, error) {
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
		workerId    *uuid.UUID
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

	if request.Params.WorkerId != nil {
		workerId = request.Params.WorkerId
	}

	opts := v1.ListTaskRunOpts{
		CreatedAfter: since,
		Statuses:     statuses,
		WorkflowIds:  workflowIds,
		WorkerId:     workerId,
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

	tasks, total, err := t.config.V1.OLAP().ListTasks(
		ctx.Request().Context(),
		tenantId,
		opts,
	)

	if err != nil {
		return nil, err
	}

	result := transformers.ToTaskSummaryMany(tasks, total, limit, offset)

	// Search for api errors to see how we handle errors in other cases
	return gen.V1TaskList200JSONResponse(
		result,
	), nil
}
