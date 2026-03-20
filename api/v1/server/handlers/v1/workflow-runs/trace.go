package workflowruns

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1WorkflowRunsService) V1WorkflowRunGetTrace(ctx echo.Context, request gen.V1WorkflowRunGetTraceRequestObject) (gen.V1WorkflowRunGetTraceResponseObject, error) {
	if !t.config.Observability.Enabled {
		return gen.V1WorkflowRunGetTrace200JSONResponse(gen.OtelSpanList{}), nil
	}

	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	tasks, err := t.config.V1.Tasks().FlattenExternalIds(ctx.Request().Context(), tenant.ID, []uuid.UUID{request.Params.RunExternalId})

	isDag := len(tasks) > 1

	var taskRunExternalID, workflowRunExternalId *uuid.UUID

	if isDag {
		workflowRunExternalId = &request.Params.RunExternalId
	} else {
		taskRunExternalID = &request.Params.RunExternalId
	}

	limit := int64(1000)
	offset := int64(0)

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	if limit < 1 {
		limit = 1000
	}

	if offset < 0 {
		offset = 0
	}

	result, err := t.config.V1.OTelCollector().ListSpansByRunExternalID(ctx.Request().Context(), tenant.ID, taskRunExternalID, workflowRunExternalId, offset, limit)
	if err != nil {
		return nil, err
	}

	return gen.V1WorkflowRunGetTrace200JSONResponse(transformers.ToV1OtelSpanList(result.Rows, nil, limit, offset, result.Total)), nil
}
