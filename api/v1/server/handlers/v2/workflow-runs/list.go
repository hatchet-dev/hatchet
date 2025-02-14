package workflowruns

import (
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *V2WorkflowRunsService) V2WorkflowRunList(ctx echo.Context, request gen.V2WorkflowRunListRequestObject) (gen.V2WorkflowRunListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	var (
		statuses = []gen.V2TaskStatus{
			gen.V2TaskStatusCANCELLED,
			gen.V2TaskStatusCOMPLETED,
			gen.V2TaskStatusFAILED,
			gen.V2TaskStatusQUEUED,
			gen.V2TaskStatusRUNNING,
		}
		since             = request.Params.Since
		workflowIds       = []uuid.UUID{}
		limit       int64 = 50
		offset      int64 = 0
	)

	if request.Params.Statuses != nil {
		if len(*request.Params.Statuses) > 0 {
			statuses = *request.Params.Statuses
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

	opts := repository.ListWorkflowRunOpts{
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

	tasks, total, err := t.config.EngineRepository.OLAP().ListWorkflowRuns(
		ctx.Request().Context(),
		tenant.ID,
		opts,
	)

	if err != nil {
		return nil, err
	}

	result := transformers.ToWorkflowRunMany(tasks, total, limit, offset)

	// Search for api errors to see how we handle errors in other cases
	return gen.V2WorkflowRunList200JSONResponse(
		result,
	), nil
}
